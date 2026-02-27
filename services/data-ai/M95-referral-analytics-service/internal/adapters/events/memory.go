package events

import (
	"context"
	"io"
	"sync"

	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/contracts"
)

type MemoryConsumer struct {
	mu     sync.Mutex
	events []contracts.EventEnvelope
}

func NewMemoryConsumer() *MemoryConsumer { return &MemoryConsumer{events: []contracts.EventEnvelope{}} }
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

type MemoryDomainPublisher struct {
	mu     sync.Mutex
	events []contracts.EventEnvelope
}

func NewMemoryDomainPublisher() *MemoryDomainPublisher {
	return &MemoryDomainPublisher{events: []contracts.EventEnvelope{}}
}
func (p *MemoryDomainPublisher) PublishDomain(_ context.Context, e contracts.EventEnvelope) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, e)
	return nil
}
func (p *MemoryDomainPublisher) Events() []contracts.EventEnvelope {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]contracts.EventEnvelope(nil), p.events...)
}

type MemoryAnalyticsPublisher struct {
	mu     sync.Mutex
	events []contracts.EventEnvelope
}

func NewMemoryAnalyticsPublisher() *MemoryAnalyticsPublisher {
	return &MemoryAnalyticsPublisher{events: []contracts.EventEnvelope{}}
}
func (p *MemoryAnalyticsPublisher) PublishAnalytics(_ context.Context, e contracts.EventEnvelope) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, e)
	return nil
}
func (p *MemoryAnalyticsPublisher) Events() []contracts.EventEnvelope {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]contracts.EventEnvelope(nil), p.events...)
}

type LoggingDLQPublisher struct{}

func NewLoggingDLQPublisher() *LoggingDLQPublisher                                       { return &LoggingDLQPublisher{} }
func (p *LoggingDLQPublisher) PublishDLQ(_ context.Context, _ contracts.DLQRecord) error { return nil }
