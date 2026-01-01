package message

import (
	"context"
	"log/slog"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
)

type ChatInterface interface {
	CheckIsMemberOfChat(ctx context.Context, chatID int64, userID int64) (bool, error)
}

//go:generate mockgen -source=message_service.go -destination=mock/message_mocks.go -package=mock
type MessageInterface interface {
	Create(ctx context.Context, chatID int64, userID int64, senderUsername string, text string) (dom.Message, error)
	EditMessage(ctx context.Context, messageID int64, newText string) error
	DeleteMessage(ctx context.Context, messageID int64) error
	CheckMessageExists(ctx context.Context, messageID int64) (bool, error)
	ListByChat(ctx context.Context, chatID int64, limit, lastMessage int) ([]dom.Message, error)
}

type MessageService struct {
	Chat    ChatInterface
	Message MessageInterface
	Logger  *slog.Logger
}

func NewMessageService(chat ChatInterface, message MessageInterface, logger *slog.Logger) *MessageService {
	return &MessageService{
		Chat:    chat,
		Message: message,
		Logger:  logger,
	}
}

func (m *MessageService) SendMessage(ctx context.Context, chatID int64, userID int64, senderUsername string, text string) (dom.Message, error) {
	isMember, err := m.Chat.CheckIsMemberOfChat(ctx, chatID, userID)
	if err != nil {
		return dom.Message{}, customerrors.ErrDatabase
	}
	if !isMember {
		return dom.Message{}, customerrors.ErrUserNotMemberOfChat
	}

	message, err := m.Message.Create(ctx, chatID, userID, senderUsername, text)
	if err != nil {
		return dom.Message{}, customerrors.ErrDatabase
	}

	return message, nil
}

func (m *MessageService) DeleteMessage(ctx context.Context, messageID int64) error {
	exists, err := m.Message.CheckMessageExists(ctx, messageID)
	if err != nil {
		return customerrors.ErrDatabase
	}
	if !exists {
		return customerrors.ErrMessageDoesNotExists
	}
	err = m.Message.DeleteMessage(ctx, messageID)
	if err != nil {
		return customerrors.ErrDatabase
	}

	return nil

}

func (m *MessageService) EditMessage(ctx context.Context, messageID int64, newText string) error {
	if newText == "" {
		m.Logger.Error("new message text is empty")
		return customerrors.ErrInvalidInput
	}
	exists, err := m.Message.CheckMessageExists(ctx, messageID)
	if err != nil {
		return customerrors.ErrDatabase
	}
	if !exists {
		return customerrors.ErrMessageDoesNotExists
	}
	if err := m.Message.EditMessage(ctx, messageID, newText); err != nil {
		return customerrors.ErrDatabase
	}
	return nil

}

func (m *MessageService) ListMessages(ctx context.Context, chatID int64, limit, lastMessage int) ([]dom.Message, error) {
	
	
	return m.Message.ListByChat(ctx, chatID, limit, lastMessage)
}
