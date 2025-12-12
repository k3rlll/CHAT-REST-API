package message

import "context"

type MessageInterface interface {
	CheckMessageExists(ctx context.Context, id int64) (bool, error)
	DeleteMessage(ctx context.Context, id int64) error
	Create(ctx context.Context, chatID int64, userID int64, senderUsername string, text string) (Message, error)
	EditMessage(ctx context.Context, messageID int64, newText string) error
	ListByChat(ctx context.Context, chatID int64) ([]Message, error)
}
