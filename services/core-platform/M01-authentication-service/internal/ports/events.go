package ports

import "context"

// EventPublisher is the outbound domain-event publish port.
// The application uses this abstraction to keep broker/client concerns in adapters.
type EventPublisher interface {
	Publish(ctx context.Context, eventType string, payload []byte) error
}
