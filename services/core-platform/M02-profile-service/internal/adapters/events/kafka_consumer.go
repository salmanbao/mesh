package events

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(brokers []string, groupID string, topics []string) (*KafkaConsumer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka consumer requires at least one broker")
	}
	if groupID == "" {
		return nil, fmt.Errorf("kafka consumer requires group id")
	}
	if len(topics) == 0 {
		return nil, fmt.Errorf("kafka consumer requires at least one topic")
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		GroupID:     groupID,
		GroupTopics: topics,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     500 * time.Millisecond,
	})
	return &KafkaConsumer{reader: reader}, nil
}

func (c *KafkaConsumer) Poll(ctx context.Context, max int) ([]Message, error) {
	if max <= 0 {
		max = 1
	}
	out := make([]Message, 0, max)
	for i := 0; i < max; i++ {
		readCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
		msg, err := c.reader.ReadMessage(readCtx)
		cancel()
		if err != nil {
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				return out, nil
			case errors.Is(err, context.Canceled):
				return out, ctx.Err()
			default:
				return out, err
			}
		}
		out = append(out, Message{
			Topic:   msg.Topic,
			Payload: msg.Value,
		})
	}
	return out, nil
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}
