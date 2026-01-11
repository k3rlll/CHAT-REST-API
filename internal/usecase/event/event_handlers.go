package event

import (
	"context"
	"encoding/json"
	"fmt"
	"main/internal/domain/events"
	"time"
)

type ChatUpdater interface {
	UpdateChatLastMessage(ctx context.Context, chatID int64, createdAt time.Time) error
	RefreshChatList(ctx context.Context, chatID int64) error
}

type EventHandlers struct {
	repo ChatUpdater
}

func NewEventHandlers(repo ChatUpdater) *EventHandlers {
	return &EventHandlers{
		repo: repo,
	}
}

func (h *EventHandlers) HandleMessageCreated(ctx context.Context, data []byte) error {
	var evt events.EventMessageCreated

	if err := json.Unmarshal(data, &evt); err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return h.repo.UpdateChatLastMessage(ctx, evt.ChatID, evt.CreatedAt)
}

func (h *EventHandlers) HandleMessageDeleted(ctx context.Context, data []byte) error {
	var evt events.EventMessageDeleted

	if err := json.Unmarshal(data, &evt); err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	return h.repo.RefreshChatList(ctx, evt.ChatID)
}
