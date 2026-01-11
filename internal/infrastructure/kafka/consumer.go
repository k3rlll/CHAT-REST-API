package kafka

import (
	"context"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type Handler func(ctx context.Context, msg []byte) error

type Consumer struct {
	reader  *kafka.Reader
	handler Handler
	logger  *slog.Logger
	topic   string
}

func NewConsumer(brokers []string, topic, groupID string, handler Handler, logger *slog.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	return &Consumer{
		reader:  reader,
		handler: handler,
		logger:  logger,
		topic:   topic,
	}
}

func (c *Consumer) StartConsuming(ctx context.Context) error {
	c.logger.Info("Kafka consumer started...", slog.String("topic", c.topic))
	defer c.reader.Close()
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.logger.Info("Kafka consumer stopped", slog.String("topic", c.topic))
				return nil
			}
			c.logger.Error("failed to fetch message from kafka", slog.String("error", err.Error()), slog.String("topic", c.topic))
			continue
		}
		if err := c.handler(ctx, msg.Value); err != nil {
			c.logger.Error("failed to process message", slog.String("error", err.Error()), slog.String("topic", c.topic))
			continue
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("failed to commit message", slog.String("error", err.Error()), slog.String("topic", c.topic))
		}
	}
}
