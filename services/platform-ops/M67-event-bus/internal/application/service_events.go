package application

import (
	"context"
	"strings"

	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/domain"
)

func (s *Service) HandleCanonicalEvent(ctx context.Context, envelope contracts.EventEnvelope) error {
	if err := validateEnvelope(envelope); err != nil {
		return err
	}
	if s.eventDedup != nil {
		dup, err := s.eventDedup.IsDuplicate(ctx, envelope.EventID, s.nowFn())
		if err != nil {
			return err
		}
		if dup {
			return nil
		}
		if err := s.eventDedup.MarkProcessed(ctx, envelope.EventID, envelope.EventType, s.nowFn().Add(s.cfg.EventDedupTTL)); err != nil {
			return err
		}
	}
	if !domain.IsCanonicalInputEvent(envelope.EventType) {
		return domain.ErrUnsupportedEventType
	}
	return nil
}

func (s *Service) FlushOutbox(context.Context) error { return nil }

func validateEnvelope(event contracts.EventEnvelope) error {
	if strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.EventType) == "" || event.OccurredAt.IsZero() {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.SourceService) == "" || strings.TrimSpace(event.TraceID) == "" || strings.TrimSpace(event.SchemaVersion) == "" {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.PartitionKeyPath) == "" || strings.TrimSpace(event.PartitionKey) == "" {
		return domain.ErrInvalidEnvelope
	}
	if len(event.Data) == 0 {
		return domain.ErrInvalidEnvelope
	}
	return nil
}
