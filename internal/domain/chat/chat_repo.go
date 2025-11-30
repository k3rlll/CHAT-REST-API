package chat

import (
	"context"
	domMessage "main/internal/domain/message"
)

type ChatRepository interface {
	GetChatDetails(ctx context.Context, chatID int64) (Chat, error)
	ListOfChats(ctx context.Context, username string) ([]Chat, error)
	CheckIfChatExists(ctx context.Context, chatID int64) (bool, error)
	DeleteChat(ctx context.Context, chatID int64) error
	CreateChat(ctx context.Context, title string, isPrivate bool, members []string) (int64, error)
	CheckIsMemberOfChat(ctx context.Context, chatID int64, username string) (bool, error)
	OpenChat(ctx context.Context, chatID int64, username string) ([]domMessage.Message, error)
	AddMembers(ctx context.Context, chatID int64, members []string) error
	UserInChat(ctx context.Context, chatID int64, username string) (bool, error)
}
