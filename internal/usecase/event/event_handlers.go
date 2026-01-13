package event

import (
	"context"
	"encoding/json"
	"fmt"
	dom "main/internal/domain/entity"
	"main/internal/domain/events"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatUpdater interface {
	//delete and update last message if needed
	UpdateChatLastMessage(ctx context.Context, chatID int64, messageText string, createdAt time.Time) error
}

type MongoMessage interface {
	GetLatestMessage(ctx context.Context, chatID int64) (dom.Message, error)
	SaveMessage(ctx context.Context, msg interface{}) (string, error)
}

type EventHandlers struct {
	repo ChatUpdater
	msg  MongoMessage
}

func NewEventHandlers(repo ChatUpdater, msg MongoMessage) *EventHandlers {
	return &EventHandlers{
		repo: repo,
		msg:  msg,
	}
}

func (h *EventHandlers) HandleMessageDeleted(ctx context.Context, data []byte) error {
	var evt events.MessageDeleted

	if err := json.Unmarshal(data, &evt); err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	message, err := h.msg.GetLatestMessage(ctx, evt.ChatID)
	if err != nil {
		return fmt.Errorf("failed to get latest message: %w", err)
	}
	if message.ID != primitive.NilObjectID {
		if err := h.repo.UpdateChatLastMessage(ctx, evt.ChatID, message.Text, message.CreatedAt); err != nil {
			return fmt.Errorf("failed to update chat last message: %w", err)
		}
	}

	return nil

}

func (h *EventHandlers) HandleMessageCreated(ctx context.Context, data []byte) error {
	var evt events.MessageCreated
	if err := json.Unmarshal(data, &evt); err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	message, err := h.msg.GetLatestMessage(ctx, evt.ChatID)
	if err != nil {
		return fmt.Errorf("failed to get latest message: %w", err)
	}

	if err := h.repo.UpdateChatLastMessage(ctx, message.ChatID, message.Text, message.CreatedAt); err != nil {
		return fmt.Errorf("failed to update chat last message: %w", err)
	}
	return nil
}
