package consumers

import (
	"context"
	dom "main/internal/domain/entity"
	"time"
)

type MessageRepository interface {
	GetLatestMessage(ctx context.Context, chatID int64) (dom.Message, error)
}

type ChatPostgresUpdater interface {
	UpdateChatLastMessage(ctx context.Context, chatID int64, messageText string, createdAt time.Time) error
}
