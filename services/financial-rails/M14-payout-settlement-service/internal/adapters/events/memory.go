package events

import (
	"context"
	"io"
	"sync"

	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/contracts"
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
	item := c.events[0]
	c.events = c.events[1:]
	return &item, nil
}

type MemoryDomainPublisher struct {
	mu     sync.Mutex
	events []contracts.EventEnvelope
}

func NewMemoryDomainPublisher() *MemoryDomainPublisher {
	return &MemoryDomainPublisher{events: []contracts.EventEnvelope{}}
}

func (p *MemoryDomainPublisher) PublishDomain(_ context.Context, event contracts.EventEnvelope) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
	return nil
}

type MemoryAnalyticsPublisher struct {
	mu     sync.Mutex
	events []contracts.EventEnvelope
}

func NewMemoryAnalyticsPublisher() *MemoryAnalyticsPublisher {
	return &MemoryAnalyticsPublisher{events: []contracts.EventEnvelope{}}
}

func (p *MemoryAnalyticsPublisher) PublishAnalytics(_ context.Context, event contracts.EventEnvelope) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
	return nil
}

type LoggingDLQPublisher struct{}

func NewLoggingDLQPublisher() *LoggingDLQPublisher {
	return &LoggingDLQPublisher{}
}

func (p *LoggingDLQPublisher) PublishDLQ(_ context.Context, _ contracts.DLQRecord) error {
	return nil
}
