package events

import "context"

type NoopConsumer struct{}

func NewNoopConsumer() *NoopConsumer {
	return &NoopConsumer{}
}

func (n *NoopConsumer) Poll(_ context.Context, _ int) ([]Message, error) {
	return nil, nil
}
