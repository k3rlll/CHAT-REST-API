package chat

import (
	"context"
	"log/slog"
)

type ChatRepository interface {
	ListOfChats(ctx context.Context) ([]Chat, error)
	CheckIfChatExists(ctx context.Context, chatID int) (bool, error)
	DeleteChat(ctx context.Context, chatID int) error
	CreateChat(ctx context.Context, name string, isPrivate bool, userIDs []int) (int, error)
	CheckIsMemberOfChat(ctx context.Context, chatID int, userID int, logger *slog.Logger) (bool, error)
}
