// internal/infrastructure/kafka/producer.go
package kafka

import (
	"context"
	"encoding/json"
	"main/internal/domain/events"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	createdWriter *kafka.Writer
	deletedWriter *kafka.Writer
}

func NewProducer(brokers []string) *Producer {
	return &Producer{
		createdWriter: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    "msg_created", // Топик для созданий
			Balancer: &kafka.LeastBytes{},
		},
		deletedWriter: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    "msg_deleted", // Топик для удалений
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Producer) SendMessageCreated(ctx context.Context, event events.MessageCreated) error {
	payload, _ := json.Marshal(event)
	return p.createdWriter.WriteMessages(ctx, kafka.Message{Value: payload})
}

func (p *Producer) SendMessageDeleted(ctx context.Context, event events.MessageDeleted) error {
	payload, _ := json.Marshal(event)
	return p.deletedWriter.WriteMessages(ctx, kafka.Message{Value: payload})
}

func (p *Producer) Close() error {
	p.createdWriter.Close()
	p.deletedWriter.Close()
	return nil
}
