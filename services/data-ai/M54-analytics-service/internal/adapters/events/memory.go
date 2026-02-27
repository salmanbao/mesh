package events

import (
	"context"
	"io"
	"sync"

	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/contracts"
)

type MemoryConsumer struct {
	mu     sync.Mutex
	events []contracts.EventEnvelope
}

func NewMemoryConsumer() *MemoryConsumer {
	return &MemoryConsumer{events: []contracts.EventEnvelope{}}
}

func (c *MemoryConsumer) Seed(events []contracts.EventEnvelope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, events...)
}

func (c *MemoryConsumer) Receive(_ context.Context) (*contracts.EventEnvelope, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.events) == 0 {
		return nil, io.EOF
	}
	e := c.events[0]
	c.events = c.events[1:]
	return &e, nil
}

type LoggingDLQPublisher struct{}

func NewLoggingDLQPublisher() *LoggingDLQPublisher {
	return &LoggingDLQPublisher{}
}

func (p *LoggingDLQPublisher) PublishDLQ(_ context.Context, _ contracts.DLQRecord) error {
	return nil
}
