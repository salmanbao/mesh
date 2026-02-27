package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/domain"
)

const internalEventReconcileAsset = "media.asset.reconcile.requested"

func (s *Service) HandleInternalEvent(ctx context.Context, envelope contracts.EventEnvelope) error {
	now := s.nowFn()
	eventID := strings.TrimSpace(envelope.EventID)
	eventClass := strings.TrimSpace(envelope.EventClass)
	if eventClass == "" {
		eventClass = domain.CanonicalEventClassDomain
	}
	switch eventClass {
	case domain.CanonicalEventClassDomain, domain.CanonicalEventClassAnalyticsOnly, domain.CanonicalEventClassOps:
	default:
		return domain.ErrUnsupportedEvent
	}

	eventType := strings.TrimSpace(envelope.EventType)
	if err := validateEnvelope(eventClass, envelope); err != nil {
		return err
	}
	if eventType != internalEventReconcileAsset {
		return domain.ErrUnsupportedEvent
	}

	if eventClass == domain.CanonicalEventClassOps {
		return nil
	}
	duplicate, err := s.eventDedup.IsDuplicate(ctx, eventID, now)
	if err != nil {
		return err
	}
	if duplicate {
		return nil
	}
	if err := s.eventDedup.MarkProcessed(ctx, eventID, eventType, now.Add(s.cfg.EventDedupTTL)); err != nil {
		return err
	}
	return nil
}

func validateEnvelope(eventClass string, envelope contracts.EventEnvelope) error {
	if strings.TrimSpace(envelope.EventID) == "" ||
		strings.TrimSpace(envelope.EventType) == "" ||
		envelope.OccurredAt.IsZero() ||
		strings.TrimSpace(envelope.SourceService) == "" ||
		strings.TrimSpace(envelope.TraceID) == "" ||
		strings.TrimSpace(envelope.SchemaVersion) == "" ||
		strings.TrimSpace(envelope.PartitionKeyPath) == "" ||
		strings.TrimSpace(envelope.PartitionKey) == "" {
		return domain.ErrUnsupportedEvent
	}

	partitionKeyPath := strings.TrimSpace(envelope.PartitionKeyPath)
	if eventClass == domain.CanonicalEventClassOps {
		if partitionKeyPath != "envelope.source_service" {
			return domain.ErrUnsupportedEvent
		}
	} else if !strings.HasPrefix(partitionKeyPath, "data.") {
		return domain.ErrUnsupportedEvent
	}

	resolved, ok := resolvePartitionKey(envelope, partitionKeyPath)
	if !ok {
		return domain.ErrUnsupportedEvent
	}
	if resolved != strings.TrimSpace(envelope.PartitionKey) {
		return domain.ErrUnsupportedEvent
	}
	return nil
}

func resolvePartitionKey(envelope contracts.EventEnvelope, path string) (string, bool) {
	if path == "envelope.source_service" {
		return strings.TrimSpace(envelope.SourceService), true
	}
	if !strings.HasPrefix(path, "data.") {
		return "", false
	}
	keyPath := strings.TrimPrefix(path, "data.")
	if keyPath == "" {
		return "", false
	}
	parts := strings.Split(keyPath, ".")
	cur := envelope.Data
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", false
		}
		switch typed := cur.(type) {
		case map[string]any:
			value, ok := typed[part]
			if !ok {
				return "", false
			}
			cur = value
		case map[string]string:
			value, ok := typed[part]
			if !ok {
				return "", false
			}
			cur = value
		default:
			return "", false
		}
	}
	return strings.TrimSpace(fmt.Sprint(cur)), true
}
