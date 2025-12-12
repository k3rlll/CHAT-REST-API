package service

import (
	"context"
	"log/slog"
	ch "main/internal/domain/chat"
	msg "main/internal/domain/message"
	"main/internal/pkg/customerrors"
)

type MessageService struct {
	Chat    ch.ChatInterface
	Message msg.MessageInterface
	Logger  *slog.Logger
}

func NewMessageService(chat ch.ChatInterface, message msg.MessageInterface, logger *slog.Logger) *MessageService {
	return &MessageService{
		Chat:    chat,
		Message: message,
		Logger:  logger,
	}
}

func (m *MessageService) Send(ctx context.Context, chatID int64, userID int64, senderUsername string, text string) (msg.Message, error) {
	isMember, err := m.Chat.CheckIsMemberOfChat(ctx, chatID, userID)
	if err != nil {
		m.Logger.Error("failed to check if user is member of chat", err.Error())
		return msg.Message{}, err
	}
	if !isMember {
		m.Logger.Error("user is not a member of the chat")
		return msg.Message{}, customerrors.ErrUserNotMemberOfChat
	}

	message, err := m.Message.Create(ctx, chatID, userID, senderUsername, text)
	if err != nil {
		m.Logger.Error("failed to create message", err.Error())
		return msg.Message{}, err
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

func (m *MessageService) Edit(ctx context.Context, messageID int64, newText string) error {
	if newText == "" {
		m.Logger.Error("new message text is empty")
		return customerrors.ErrMessageIsEmpty
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

func (m *MessageService) List(ctx context.Context, chatID int64) ([]msg.Message, error) {
	return m.Message.ListByChat(ctx, chatID)
}
