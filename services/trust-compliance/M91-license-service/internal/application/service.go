package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M91-license-service/internal/domain"
	"github.com/viralforge/mesh/services/trust-compliance/M91-license-service/internal/ports"
)

type Service struct {
	cfg             Config
	licenses        ports.LicenseRepository
	activations     ports.ActivationRepository
	revocations     ports.RevocationRepository
	configs         ports.ProductConfigRepository
	idempotency     ports.IdempotencyRepository
	nowFn           func() time.Time
	mu              sync.Mutex
	validationByIP  map[string][]time.Time
	activationByKey map[string][]time.Time
}

type Dependencies struct {
	Config      Config
	Licenses    ports.LicenseRepository
	Activations ports.ActivationRepository
	Revocations ports.RevocationRepository
	Configs     ports.ProductConfigRepository
	Idempotency ports.IdempotencyRepository
}

var idCounter uint64

func NewService(deps Dependencies) *Service {
	s := &Service{
		cfg:             deps.Config,
		licenses:        deps.Licenses,
		activations:     deps.Activations,
		revocations:     deps.Revocations,
		configs:         deps.Configs,
		idempotency:     deps.Idempotency,
		nowFn:           time.Now().UTC,
		validationByIP:  map[string][]time.Time{},
		activationByKey: map[string][]time.Time{},
	}
	if s.cfg.IdempotencyTTL == 0 {
		s.cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	return s
}

func (s *Service) ListLicenses(ctx context.Context, actor Actor) ([]domain.License, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	return s.licenses.List(ctx)
}

func (s *Service) Validate(ctx context.Context, actor Actor, licenseKey string) (map[string]any, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	key := strings.TrimSpace(licenseKey)
	if key == "" {
		return nil, domain.ErrInvalidInput
	}
	if err := s.checkValidationRate(actor.ClientIP); err != nil {
		return nil, err
	}
	license, err := s.licenses.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	revocations, _ := s.revocations.ListByLicense(ctx, license.ID)
	valid := license.Status == "active" && len(revocations) == 0
	return map[string]any{
		"license_id":     license.ID,
		"license_key":    license.LicenseKey,
		"valid":          valid,
		"status":         license.Status,
		"product_id":     license.ProductID,
		"transaction_id": license.TransactionID,
	}, nil
}

func (s *Service) Activate(ctx context.Context, actor Actor, input ActivateInput) (map[string]any, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return nil, domain.ErrIdempotencyRequired
	}
	if strings.TrimSpace(input.LicenseKey) == "" || strings.TrimSpace(input.DeviceID) == "" || strings.TrimSpace(input.DeviceFingerprint) == "" {
		return nil, domain.ErrInvalidInput
	}
	requestHash := hashJSON(input)
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return nil, err
	} else if ok {
		var out map[string]any
		_ = json.Unmarshal(rec, &out)
		return out, nil
	}
	license, err := s.licenses.GetByKey(ctx, input.LicenseKey)
	if err != nil {
		return nil, err
	}
	if license.Status != "active" {
		return nil, domain.ErrConflict
	}
	if err := s.checkActivationRate(license.LicenseKey); err != nil {
		return nil, err
	}
	cfg, err := s.configs.GetByProductID(ctx, license.ProductID)
	if err != nil {
		return nil, err
	}
	items, _ := s.activations.ListByLicense(ctx, license.ID)
	activeCount := 0
	for _, item := range items {
		if item.Status == "active" {
			if item.DeviceID == input.DeviceID {
				out := map[string]any{"license_id": license.ID, "license_key": license.LicenseKey, "activation_status": "active", "device_id": item.DeviceID}
				_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, out)
				return out, nil
			}
			activeCount++
		}
	}
	limit := cfg.MaxActivations
	if limit <= 0 {
		limit = license.MaxActivations
	}
	if activeCount >= limit {
		return nil, domain.ErrConflict
	}
	now := s.nowFn()
	activation := domain.Activation{
		ID:                newID("act", now),
		LicenseID:         license.ID,
		DeviceID:          strings.TrimSpace(input.DeviceID),
		DeviceFingerprint: strings.TrimSpace(input.DeviceFingerprint),
		IPHash:            hashString(actor.ClientIP),
		ActivatedAt:       now,
		Status:            "active",
	}
	if err := s.activations.Add(ctx, activation); err != nil {
		return nil, err
	}
	license.UpdatedAt = now
	_ = s.licenses.Update(ctx, license)
	out := map[string]any{"license_id": license.ID, "license_key": license.LicenseKey, "activation_status": "active", "device_id": activation.DeviceID}
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, out)
	return out, nil
}

func (s *Service) Deactivate(ctx context.Context, actor Actor, input DeactivateInput) (map[string]any, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return nil, domain.ErrIdempotencyRequired
	}
	if strings.TrimSpace(input.LicenseKey) == "" || strings.TrimSpace(input.DeviceID) == "" {
		return nil, domain.ErrInvalidInput
	}
	requestHash := hashJSON(input)
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return nil, err
	} else if ok {
		var out map[string]any
		_ = json.Unmarshal(rec, &out)
		return out, nil
	}
	license, err := s.licenses.GetByKey(ctx, input.LicenseKey)
	if err != nil {
		return nil, err
	}
	items, _ := s.activations.ListByLicense(ctx, license.ID)
	for _, item := range items {
		if item.DeviceID == strings.TrimSpace(input.DeviceID) && item.Status == "active" {
			now := s.nowFn()
			item.Status = "inactive"
			item.DeactivatedAt = now
			if err := s.activations.Update(ctx, item); err != nil {
				return nil, err
			}
			out := map[string]any{"license_id": license.ID, "license_key": license.LicenseKey, "deactivation_status": "inactive", "device_id": item.DeviceID}
			_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, out)
			return out, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (s *Service) Export(ctx context.Context, actor Actor, input ExportInput) (domain.ExportRecord, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ExportRecord{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ExportRecord{}, domain.ErrIdempotencyRequired
	}
	format := strings.ToLower(strings.TrimSpace(input.Format))
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		return domain.ExportRecord{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(input)
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ExportRecord{}, err
	} else if ok {
		var out domain.ExportRecord
		_ = json.Unmarshal(rec, &out)
		return out, nil
	}
	licenses, err := s.licenses.List(ctx)
	if err != nil {
		return domain.ExportRecord{}, err
	}
	out := domain.ExportRecord{Format: format, GeneratedAt: s.nowFn(), Rows: licenses}
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, out)
	return out, nil
}

func (s *Service) checkValidationRate(ip string) error {
	if strings.TrimSpace(ip) == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.nowFn()
	windowStart := now.Add(-1 * time.Hour)
	items := pruneTimes(s.validationByIP[ip], windowStart)
	if len(items) >= 5 {
		s.validationByIP[ip] = items
		return domain.ErrRateLimited
	}
	items = append(items, now)
	s.validationByIP[ip] = items
	return nil
}

func (s *Service) checkActivationRate(licenseKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.nowFn()
	windowStart := now.Add(-24 * time.Hour)
	key := strings.ToUpper(strings.TrimSpace(licenseKey))
	items := pruneTimes(s.activationByKey[key], windowStart)
	if len(items) >= 10 {
		s.activationByKey[key] = items
		return domain.ErrRateLimited
	}
	items = append(items, now)
	s.activationByKey[key] = items
	return nil
}

func pruneTimes(items []time.Time, min time.Time) []time.Time {
	out := make([]time.Time, 0, len(items))
	for _, item := range items {
		if item.After(min) {
			out = append(out, item)
		}
	}
	return out
}

func (s *Service) getIdempotent(ctx context.Context, key, hash string) ([]byte, bool, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil, false, nil
	}
	rec, err := s.idempotency.Get(ctx, key, s.nowFn())
	if err != nil || rec == nil {
		return nil, false, err
	}
	if rec.RequestHash != hash {
		return nil, false, domain.ErrIdempotencyConflict
	}
	return rec.Response, true, nil
}

func (s *Service) completeIdempotent(ctx context.Context, key, requestHash string, payload any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	raw, _ := json.Marshal(payload)
	return s.idempotency.Upsert(ctx, domain.IdempotencyRecord{Key: key, RequestHash: requestHash, Response: raw, ExpiresAt: s.nowFn().Add(s.cfg.IdempotencyTTL)})
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func hashString(v string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(v)))
	return hex.EncodeToString(sum[:])
}

func newID(prefix string, now time.Time) string {
	n := atomic.AddUint64(&idCounter, 1)
	return prefix + "-" + shortID(now.UnixNano()+int64(n))
}

func shortID(v int64) string {
	if v < 0 {
		v = -v
	}
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 16)
	for v > 0 {
		buf = append(buf, chars[v%int64(len(chars))])
		v /= int64(len(chars))
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
