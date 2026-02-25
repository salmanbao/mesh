package events

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	writer       *kafka.Writer
	topicByEvent map[string]string
}

func NewKafkaPublisher(brokers []string, topicByEvent map[string]string) (*KafkaPublisher, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka publisher requires at least one broker")
	}
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			RequiredAcks: kafka.RequireAll,
			Balancer:     &kafka.Hash{},
		},
		topicByEvent: topicByEvent,
	}, nil
}

func (p *KafkaPublisher) Publish(ctx context.Context, eventType string, payload []byte, partitionKey string) error {
	topic := eventType
	if mapped, ok := p.topicByEvent[eventType]; ok && mapped != "" {
		topic = mapped
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(partitionKey),
		Value: payload,
		Time:  time.Now().UTC(),
	})
}

func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}
