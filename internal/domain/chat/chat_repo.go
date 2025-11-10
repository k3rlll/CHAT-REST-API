package chat

import (
	"context"
	"log/slog"
)

type ChatRepository interface {
	CreateChat(ctx context.Context, name string, isPrivate bool, userIDs []int) (string, int, error)
	CheckIsMemberOfChat(ctx context.Context, chatID int, userID int, logger *slog.Logger) (bool, error)
}
