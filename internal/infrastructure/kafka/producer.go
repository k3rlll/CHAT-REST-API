package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"main/internal/domain/events"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &Producer{
		writer: writer,
	}
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

func (p *Producer) SendMessageCreated(ctx context.Context, event events.EventMessageCreated) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event : %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", event.ChatID)),
		Value: payload,
		Time:  time.Now(),
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}
	return nil
}
