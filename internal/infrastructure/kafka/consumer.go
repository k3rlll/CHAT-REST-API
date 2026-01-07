package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"main/internal/domain/events"
	"time"

	"github.com/segmentio/kafka-go"
)

type PostgresUpdater interface {
	UpdateChatLastMessage(ctx context.Context, chatID int64, at time.Time) error
}

type Consumer struct {
	reader *kafka.Reader
	pgRepo PostgresUpdater
	logger *slog.Logger
}

func NewConsumer(brokers []string, topic, groupID string, pgRepo PostgresUpdater, logger *slog.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	return &Consumer{
		reader: reader,
		pgRepo: pgRepo,
		logger: logger,
	}
}

func (c *Consumer) StartConsuming(ctx context.Context) error {
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

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var event events.EventMessageCreated

	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal kafka message: %w", err)
	}

	return c.pgRepo.UpdateChatLastMessage(ctx, event.ChatID, event.CreatedAt)
}
