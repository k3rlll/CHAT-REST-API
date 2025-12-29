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

type MessageInterface interface {
	Create(ctx context.Context, chatID int64, userID int64, senderUsername string, text string) (dom.Message, error)
	EditMessage(ctx context.Context, messageID int64, newText string) error
	DeleteMessage(ctx context.Context, messageID int64) error
	CheckMessageExists(ctx context.Context, messageID int64) (bool, error)
	ListByChat(ctx context.Context, chatID int64) ([]dom.Message, error)
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
		m.Logger.Error("failed to check if user is member of chat", err.Error())
		return dom.Message{}, err
	}
	if !isMember {
		m.Logger.Error("user is not a member of the chat")
		return dom.Message{}, customerrors.ErrUserNotMemberOfChat
	}

	message, err := m.Message.Create(ctx, chatID, userID, senderUsername, text)
	if err != nil {
		m.Logger.Error("failed to create message", err.Error())
		return dom.Message{}, err
	}

	return message, nil
}

func (m *MessageService) DeleteMessage(ctx context.Context, messageID int64) error {
	exists, err := m.Message.CheckMessageExists(ctx, messageID)
	if err != nil {
		m.Logger.Error(err.Error())
		return err
	}
	if !exists {
		m.Logger.Error("the message does not exists")
		return customerrors.ErrMessageDoesNotExists
	}
	err = m.Message.DeleteMessage(ctx, messageID)
	if err != nil {
		m.Logger.Error("failed to delete message", err.Error())
		return err
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
		m.Logger.Error("failed to check if the message exists", err.Error())
		return err
	}
	if !exists {
		m.Logger.Error("the message does not exists")
		return customerrors.ErrMessageDoesNotExists
	}
	if err := m.Message.EditMessage(ctx, messageID, newText); err != nil {
		m.Logger.Error("failed to edit message", err.Error())
		return err
	}
	return nil

}

func (m *MessageService) ListMessages(ctx context.Context, chatID int64) ([]dom.Message, error) {
	return m.Message.ListByChat(ctx, chatID)
}
