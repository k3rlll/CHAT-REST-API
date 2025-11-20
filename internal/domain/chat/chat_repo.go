package chat

import (
	"context"
	domMessage "main/internal/domain/message"
)

type ChatRepository interface {
	GetChatDetails(ctx context.Context, chatID int64) (Chat, error)
	ListOfChats(ctx context.Context) ([]Chat, error)
	CheckIfChatExists(ctx context.Context, chatID int64) (bool, error)
	DeleteChat(ctx context.Context, chatID int64) error
	CreateChat(ctx context.Context, name string, isPrivate bool, userIDs []int64) (int64, error)
	CheckIsMemberOfChat(ctx context.Context, chatID int64, userID int64) (bool, error)
	OpenChat(ctx context.Context, chatID int64, userID int64) ([]domMessage.Message, error)
	AddMembers(ctx context.Context, chatID int64, members []int64) error
	UserInChat(ctx context.Context, chatID int64, userID int64) (bool, error)
}
