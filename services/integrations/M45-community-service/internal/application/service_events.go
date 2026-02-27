package application

import (
	"context"
	"strings"

	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/domain"
)

func (s *Service) HandleCanonicalEvent(_ context.Context, envelope contracts.EventEnvelope) error {
	if err := validateEnvelope(envelope); err != nil {
		return err
	}
	if !domain.IsCanonicalInputEvent(envelope.EventType) {
		return domain.ErrUnsupportedEventType
	}
	return nil
}

func validateEnvelope(event contracts.EventEnvelope) error {
	if strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.EventType) == "" || event.OccurredAt.IsZero() {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.SourceService) == "" || strings.TrimSpace(event.TraceID) == "" || strings.TrimSpace(event.SchemaVersion) == "" {
		return domain.ErrInvalidEnvelope
	}
	if len(event.Data) == 0 {
		return domain.ErrInvalidEnvelope
	}
	return nil
}
