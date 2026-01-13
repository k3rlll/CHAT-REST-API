package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"main/internal/domain/events"

	"github.com/segmentio/kafka-go"
)

type ConsumerDeleted struct {
	reader  *kafka.Reader
	chat    ChatPostgresUpdater
	message MessageRepository
	logger  *slog.Logger
}

func NewConsumerDeleted(brokers []string, topic, groupID string, chat ChatPostgresUpdater, logger *slog.Logger) *ConsumerDeleted {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	return &ConsumerDeleted{
		reader: reader,
		chat:   chat,
		logger: logger,
	}
}

func (c *ConsumerDeleted) StartConsumerDeleted(ctx context.Context) error {
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

func (c *ConsumerDeleted) processMessage(ctx context.Context, msg kafka.Message) error {
	var EventMessageCreated events.MessageCreated

	if err := json.Unmarshal(msg.Value, &EventMessageCreated); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)

	}
	res, err := c.message.GetLatestMessage(ctx, EventMessageCreated.ChatID)
	if err != nil {
		return fmt.Errorf("failed to get latest message: %w", err)
	}
	return c.chat.UpdateChatLastMessage(ctx, EventMessageCreated.ChatID, res.Text, res.CreatedAt)
}
