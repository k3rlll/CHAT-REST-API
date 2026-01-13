package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"main/internal/domain/events"

	"github.com/segmentio/kafka-go"
)

type ConsumerCreated struct {
	reader *kafka.Reader
	chat   ChatPostgresUpdater
	logger *slog.Logger
}

func NewConsumerCreated(brokers []string, topic, groupID string, chat ChatPostgresUpdater, logger *slog.Logger) *ConsumerCreated {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	return &ConsumerCreated{
		reader: reader,
		chat:   chat,
		logger: logger,
	}
}

func (c *ConsumerCreated) StartConsumerCreated(ctx context.Context) error {
	c.logger.Info("Kafka consumer started...")
	defer c.reader.Close()
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.logger.Info("Kafka consumer stopped")
				return nil
			}
			c.logger.Error("failed to fetch message from kafka", slog.String("error", err.Error()))
			continue
		}
		if err := c.processMessage(ctx, msg); err != nil {
			c.logger.Error("failed to process message", slog.String("error", err.Error()))
			continue
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("failed to commit message", slog.String("error", err.Error()))
		}
	}
}

func (c *ConsumerCreated) processMessage(ctx context.Context, msg kafka.Message) error {
	var EventMessageCreated events.MessageCreated

	if err := json.Unmarshal(msg.Value, &EventMessageCreated); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)

	}

	return c.chat.UpdateChatLastMessage(ctx, EventMessageCreated.ChatID, EventMessageCreated.Text, EventMessageCreated.CreatedAt)
}
