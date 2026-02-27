package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/ports"
)

func (s *Service) CreateTransaction(ctx context.Context, actor Actor, input CreateTransactionInput) (domain.Transaction, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Transaction{}, domain.ErrUnauthorized
	}
	if actor.Role != "admin" && actor.SubjectID != input.UserID {
		return domain.Transaction{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Transaction{}, domain.ErrIdempotencyRequired
	}
	return s.createTransactionWithKey(ctx, input, actor.IdempotencyKey)
}

func (s *Service) GetTransaction(ctx context.Context, actor Actor, transactionID string) (domain.Transaction, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Transaction{}, domain.ErrUnauthorized
	}
	transaction, err := s.transactions.GetByID(ctx, transactionID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if actor.Role != "admin" && actor.SubjectID != transaction.UserID {
		return domain.Transaction{}, domain.ErrForbidden
	}
	return transaction, nil
}

func (s *Service) ListTransactions(ctx context.Context, actor Actor, query ports.TransactionListQuery) (ListTransactionsOutput, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return ListTransactionsOutput{}, domain.ErrUnauthorized
	}
	if actor.Role != "admin" {
		query.UserID = actor.SubjectID
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	items, total, err := s.transactions.List(ctx, query)
	if err != nil {
		return ListTransactionsOutput{}, err
	}
	return ListTransactionsOutput{
		Items: items,
		Pagination: contracts.Pagination{
			Limit:  query.Limit,
			Offset: query.Offset,
			Total:  total,
		},
	}, nil
}

func (s *Service) GetBalance(ctx context.Context, actor Actor, userID string) (domain.UserBalance, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.UserBalance{}, domain.ErrUnauthorized
	}
	if actor.Role != "admin" && actor.SubjectID != userID {
		return domain.UserBalance{}, domain.ErrForbidden
	}
	return s.balances.GetOrCreate(ctx, userID)
}

func (s *Service) CreateRefund(ctx context.Context, actor Actor, input CreateRefundInput) (domain.Refund, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Refund{}, domain.ErrUnauthorized
	}
	if actor.Role != "admin" && actor.SubjectID != input.UserID {
		return domain.Refund{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Refund{}, domain.ErrIdempotencyRequired
	}
	if err := domain.ValidateRefundInput(input.TransactionID, input.UserID, input.Reason, input.Amount); err != nil {
		return domain.Refund{}, err
	}

	transaction, err := s.transactions.GetByID(ctx, input.TransactionID)
	if err != nil {
		return domain.Refund{}, err
	}
	if actor.Role != "admin" && transaction.UserID != actor.SubjectID {
		return domain.Refund{}, domain.ErrForbidden
	}
	if transaction.Status == domain.TransactionStatusRefunded {
		return domain.Refund{}, domain.ErrConflict
	}
	if transaction.Status != domain.TransactionStatusSucceeded {
		return domain.Refund{}, domain.ErrConflict
	}

	requestHash := hashPayload(input)
	now := s.nowFn()
	existing, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now)
	if err != nil {
		return domain.Refund{}, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			return domain.Refund{}, domain.ErrIdempotencyConflict
		}
		var cached domain.Refund
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.Refund{}, err
		}
		return cached, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.Refund{}, err
	}

	refund := domain.Refund{
		RefundID:       uuid.NewString(),
		TransactionID:  transaction.TransactionID,
		UserID:         input.UserID,
		Amount:         input.Amount,
		Currency:       transaction.Currency,
		Reason:         input.Reason,
		IdempotencyKey: actor.IdempotencyKey,
		CreatedAt:      now,
	}
	if err := s.refunds.Create(ctx, refund); err != nil {
		return domain.Refund{}, err
	}

	transaction.Status = domain.TransactionStatusRefunded
	transaction.FailureReason = input.Reason
	transaction.RefundedAt = &now
	transaction.UpdatedAt = now
	if err := s.transactions.Update(ctx, transaction); err != nil {
		return domain.Refund{}, err
	}

	balance, err := s.balances.GetOrCreate(ctx, transaction.UserID)
	if err != nil {
		return domain.Refund{}, err
	}
	balance.PendingBalance -= input.Amount
	if balance.PendingBalance < 0 {
		balance.NegativeBalance += -balance.PendingBalance
		balance.PendingBalance = 0
	}
	balance.LastTransactionID = transaction.TransactionID
	balance.UpdatedAt = now
	if err := s.balances.Upsert(ctx, balance); err != nil {
		return domain.Refund{}, err
	}

	if err := s.enqueueDomainTransactionRefunded(ctx, transaction, refund); err != nil {
		return domain.Refund{}, err
	}
	if err := s.FlushOutbox(ctx); err != nil {
		return domain.Refund{}, err
	}

	payload, err := json.Marshal(refund)
	if err != nil {
		return domain.Refund{}, err
	}
	if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 201, payload, s.nowFn()); err != nil {
		return domain.Refund{}, err
	}
	return refund, nil
}

func (s *Service) HandleProviderWebhook(ctx context.Context, input HandleWebhookInput) (domain.Transaction, error) {
	if err := domain.ValidateWebhookInput(input.WebhookID, input.Provider, input.EventType); err != nil {
		return domain.Transaction{}, err
	}
	now := s.nowFn()

	seen, err := s.webhooks.IsDuplicate(ctx, input.WebhookID, now)
	if err != nil {
		return domain.Transaction{}, err
	}
	if seen {
		if input.TransactionID != "" {
			return s.transactions.GetByID(ctx, input.TransactionID)
		}
		if input.ProviderTransactionID != "" {
			return s.transactions.GetByProviderTransactionID(ctx, input.ProviderTransactionID)
		}
		return domain.Transaction{}, domain.ErrWebhookAlreadyHandled
	}

	idempotencyKey := "webhook:" + input.WebhookID
	requestHash := hashPayload(input)
	existing, err := s.idempotency.Get(ctx, idempotencyKey, now)
	if err != nil {
		return domain.Transaction{}, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			return domain.Transaction{}, domain.ErrIdempotencyConflict
		}
		var cached domain.Transaction
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.Transaction{}, err
		}
		return cached, nil
	}
	if err := s.idempotency.Reserve(ctx, idempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.Transaction{}, err
	}

	transaction, err := s.lookupWebhookTransaction(ctx, input)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.Transaction{}, err
	}
	if errors.Is(err, domain.ErrNotFound) {
		if strings.TrimSpace(input.UserID) == "" || input.Amount <= 0 {
			return domain.Transaction{}, domain.ErrNotFound
		}
		currency := strings.ToUpper(strings.TrimSpace(input.Currency))
		if currency == "" {
			currency = s.cfg.DefaultCurrency
		}
		transaction = domain.Transaction{
			TransactionID:         uuid.NewString(),
			UserID:                input.UserID,
			CampaignID:            "unknown",
			ProductID:             "unknown",
			Provider:              domain.PaymentProvider(strings.ToLower(strings.TrimSpace(input.Provider))),
			ProviderTransactionID: input.ProviderTransactionID,
			Amount:                input.Amount,
			Currency:              currency,
			Status:                domain.TransactionStatusPending,
			IdempotencyKey:        idempotencyKey,
			CreatedAt:             now,
			UpdatedAt:             now,
		}
		if err := s.transactions.Create(ctx, transaction); err != nil {
			return domain.Transaction{}, err
		}
	}

	switch {
	case domain.IsSuccessWebhook(input.EventType):
		at := s.nowFn()
		if transaction.Status != domain.TransactionStatusSucceeded {
			transaction.Status = domain.TransactionStatusSucceeded
			transaction.FailureReason = ""
			transaction.SucceededAt = &at
			transaction.UpdatedAt = at
			if err := s.transactions.Update(ctx, transaction); err != nil {
				return domain.Transaction{}, err
			}
			balance, err := s.balances.GetOrCreate(ctx, transaction.UserID)
			if err != nil {
				return domain.Transaction{}, err
			}
			balance.PendingBalance += transaction.Amount
			balance.LastTransactionID = transaction.TransactionID
			balance.UpdatedAt = at
			if err := s.balances.Upsert(ctx, balance); err != nil {
				return domain.Transaction{}, err
			}
			if err := s.enqueueDomainTransactionSucceeded(ctx, transaction); err != nil {
				return domain.Transaction{}, err
			}
		}
	case domain.IsFailureWebhook(input.EventType):
		at := s.nowFn()
		if transaction.Status != domain.TransactionStatusFailed {
			transaction.Status = domain.TransactionStatusFailed
			transaction.FailureReason = strings.TrimSpace(input.Reason)
			transaction.FailedAt = &at
			transaction.UpdatedAt = at
			if err := s.transactions.Update(ctx, transaction); err != nil {
				return domain.Transaction{}, err
			}
			if err := s.enqueueDomainTransactionFailed(ctx, transaction); err != nil {
				return domain.Transaction{}, err
			}
		}
	case domain.IsRefundWebhook(input.EventType):
		at := s.nowFn()
		refundAmount := input.Amount
		if refundAmount <= 0 {
			refundAmount = transaction.Amount
		}
		refund := domain.Refund{
			RefundID:       uuid.NewString(),
			TransactionID:  transaction.TransactionID,
			UserID:         transaction.UserID,
			Amount:         refundAmount,
			Currency:       transaction.Currency,
			Reason:         strings.TrimSpace(input.Reason),
			IdempotencyKey: idempotencyKey,
			CreatedAt:      at,
		}
		if err := s.refunds.Create(ctx, refund); err != nil {
			return domain.Transaction{}, err
		}
		transaction.Status = domain.TransactionStatusRefunded
		transaction.RefundedAt = &at
		transaction.UpdatedAt = at
		if err := s.transactions.Update(ctx, transaction); err != nil {
			return domain.Transaction{}, err
		}
		balance, err := s.balances.GetOrCreate(ctx, transaction.UserID)
		if err != nil {
			return domain.Transaction{}, err
		}
		balance.PendingBalance -= refund.Amount
		if balance.PendingBalance < 0 {
			balance.NegativeBalance += -balance.PendingBalance
			balance.PendingBalance = 0
		}
		balance.LastTransactionID = transaction.TransactionID
		balance.UpdatedAt = at
		if err := s.balances.Upsert(ctx, balance); err != nil {
			return domain.Transaction{}, err
		}
		if err := s.enqueueDomainTransactionRefunded(ctx, transaction, refund); err != nil {
			return domain.Transaction{}, err
		}
	default:
		return domain.Transaction{}, domain.ErrUnsupportedEventType
	}

	if err := s.webhooks.MarkProcessed(ctx, domain.Webhook{
		WebhookID:             input.WebhookID,
		Provider:              input.Provider,
		EventType:             input.EventType,
		ProviderEventID:       input.ProviderEventID,
		ProviderTransactionID: input.ProviderTransactionID,
		TransactionID:         transaction.TransactionID,
		Status:                string(transaction.Status),
		ReceivedAt:            now,
		ProcessedAt:           s.nowFn(),
	}, now.Add(s.cfg.EventDedupTTL)); err != nil {
		return domain.Transaction{}, err
	}

	if err := s.FlushOutbox(ctx); err != nil {
		return domain.Transaction{}, err
	}
	payload, err := json.Marshal(transaction)
	if err != nil {
		return domain.Transaction{}, err
	}
	if err := s.idempotency.Complete(ctx, idempotencyKey, 200, payload, s.nowFn()); err != nil {
		return domain.Transaction{}, err
	}
	return transaction, nil
}

func (s *Service) createTransactionWithKey(ctx context.Context, input CreateTransactionInput, idempotencyKey string) (domain.Transaction, error) {
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = s.cfg.DefaultCurrency
	}
	if err := domain.ValidateCreateTransactionInput(input.UserID, input.CampaignID, input.ProductID, input.Provider, input.Amount, currency); err != nil {
		return domain.Transaction{}, err
	}

	if _, err := s.auth.GetUser(ctx, input.UserID); err != nil {
		return domain.Transaction{}, fmt.Errorf("auth lookup: %w", err)
	}
	if err := s.campaign.EnsureCampaignAccessible(ctx, input.CampaignID, input.UserID); err != nil {
		return domain.Transaction{}, fmt.Errorf("campaign access check: %w", err)
	}
	if err := s.contentLibrary.EnsureProductLicensed(ctx, input.ProductID, input.UserID); err != nil {
		return domain.Transaction{}, fmt.Errorf("product licensing check: %w", err)
	}
	if err := s.escrow.EnsureFundingSource(ctx, input.UserID, input.Amount, currency); err != nil {
		return domain.Transaction{}, fmt.Errorf("escrow funding check: %w", err)
	}
	feeRate, err := s.feeEngine.GetFeeRate(ctx, input.TrafficSource, input.UserTier)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("fee lookup: %w", err)
	}
	if err := s.product.EnsureProductActive(ctx, input.ProductID); err != nil {
		return domain.Transaction{}, fmt.Errorf("product status check: %w", err)
	}

	requestHash := hashPayload(input)
	now := s.nowFn()
	existing, err := s.idempotency.Get(ctx, idempotencyKey, now)
	if err != nil {
		return domain.Transaction{}, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			return domain.Transaction{}, domain.ErrIdempotencyConflict
		}
		var cached domain.Transaction
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.Transaction{}, err
		}
		return cached, nil
	}
	if err := s.idempotency.Reserve(ctx, idempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.Transaction{}, err
	}

	providerTransactionID := strings.TrimSpace(input.ProviderTransactionID)
	if providerTransactionID == "" {
		providerTransactionID = "prov-" + uuid.NewString()
	}
	transaction := domain.Transaction{
		TransactionID:         uuid.NewString(),
		UserID:                input.UserID,
		CampaignID:            input.CampaignID,
		ProductID:             input.ProductID,
		Provider:              input.Provider,
		ProviderTransactionID: providerTransactionID,
		Amount:                input.Amount,
		Currency:              currency,
		PlatformFeeRate:       feeRate,
		Status:                domain.TransactionStatusPending,
		IdempotencyKey:        idempotencyKey,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	switch {
	case input.Amount > s.cfg.MaximumAmount:
		at := s.nowFn()
		transaction.Status = domain.TransactionStatusFailed
		transaction.FailureReason = "amount_exceeds_limit"
		transaction.FailedAt = &at
		transaction.UpdatedAt = at
	default:
		at := s.nowFn()
		transaction.Status = domain.TransactionStatusSucceeded
		transaction.SucceededAt = &at
		transaction.UpdatedAt = at
	}

	if err := s.transactions.Create(ctx, transaction); err != nil {
		return domain.Transaction{}, err
	}

	if transaction.Status == domain.TransactionStatusSucceeded {
		balance, err := s.balances.GetOrCreate(ctx, input.UserID)
		if err != nil {
			return domain.Transaction{}, err
		}
		net := transaction.Amount * (1 - transaction.PlatformFeeRate)
		if net < 0 {
			net = 0
		}
		balance.PendingBalance += net
		balance.LastTransactionID = transaction.TransactionID
		balance.UpdatedAt = s.nowFn()
		if err := s.balances.Upsert(ctx, balance); err != nil {
			return domain.Transaction{}, err
		}
		if err := s.enqueueDomainTransactionSucceeded(ctx, transaction); err != nil {
			return domain.Transaction{}, err
		}
	} else {
		if err := s.enqueueDomainTransactionFailed(ctx, transaction); err != nil {
			return domain.Transaction{}, err
		}
	}

	if err := s.FlushOutbox(ctx); err != nil {
		return domain.Transaction{}, err
	}
	payload, err := json.Marshal(transaction)
	if err != nil {
		return domain.Transaction{}, err
	}
	if err := s.idempotency.Complete(ctx, idempotencyKey, 201, payload, s.nowFn()); err != nil {
		return domain.Transaction{}, err
	}
	return transaction, nil
}

func (s *Service) lookupWebhookTransaction(ctx context.Context, input HandleWebhookInput) (domain.Transaction, error) {
	if strings.TrimSpace(input.TransactionID) != "" {
		return s.transactions.GetByID(ctx, input.TransactionID)
	}
	if strings.TrimSpace(input.ProviderTransactionID) != "" {
		return s.transactions.GetByProviderTransactionID(ctx, input.ProviderTransactionID)
	}
	return domain.Transaction{}, domain.ErrNotFound
}

func hashPayload(value interface{}) string {
	blob, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}
