package ports

import "context"

type EventPublisher interface {
	Publish(ctx context.Context, eventType string, payload []byte, partitionKey string) error
}
