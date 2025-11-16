package chat

import (
	"context"
	domMessage "main/internal/domain/message"
)

type ChatRepository interface {
	GetChatDetails(ctx context.Context, chatID int) (Chat, error)
	ListOfChats(ctx context.Context) ([]Chat, error)
	CheckIfChatExists(ctx context.Context, chatID int) (bool, error)
	DeleteChat(ctx context.Context, chatID int) error
	CreateChat(ctx context.Context, name string, isPrivate bool, userIDs []int) (int, error)
	CheckIsMemberOfChat(ctx context.Context, chatID int, userID int) (bool, error)
	OpenChat(ctx context.Context, chatID int, userID int) ([]domMessage.Message, error)
	AddMembers(ctx context.Context, chatID int, members []int) error
	UserInChat(ctx context.Context, chatID int, userID int) (bool, error)
}
