package events

import (
	"context"
	"errors"
	"io"
	"slices"
	"sync"

	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/contracts"
)

type MemoryJobQueue struct {
	mu    sync.Mutex
	items []string
}

func NewMemoryJobQueue() *MemoryJobQueue {
	return &MemoryJobQueue{items: make([]string, 0, 128)}
}

func (q *MemoryJobQueue) Enqueue(_ context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, jobID)
	return nil
}

func (q *MemoryJobQueue) Dequeue(_ context.Context) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return "", io.EOF
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, nil
}

func (q *MemoryJobQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

type MemoryDLQPublisher struct {
	mu      sync.Mutex
	records []contracts.QueueDLQRecord
}

func NewMemoryDLQPublisher() *MemoryDLQPublisher {
	return &MemoryDLQPublisher{records: make([]contracts.QueueDLQRecord, 0, 32)}
}

func (p *MemoryDLQPublisher) Publish(_ context.Context, record contracts.QueueDLQRecord) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.records = append(p.records, record)
	return nil
}

func (p *MemoryDLQPublisher) Records() []contracts.QueueDLQRecord {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.records)
}

func IsIdleError(err error) bool {
	return errors.Is(err, io.EOF)
}
