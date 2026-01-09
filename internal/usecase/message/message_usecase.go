package message

import (
	"context"
	"fmt"
	"log/slog"
	dom "main/internal/domain/entity"
	"main/internal/domain/events"
	"main/internal/pkg/customerrors"
	"time"
)

type ChatInterface interface {
	CheckIsMemberOfChat(ctx context.Context, chatID int64, userID int64) (bool, error)
}

//go:generate mockgen -source=message_usecase.go -destination=mock/message_mocks.go -package=mock

// type MessageInterface interface {
// 	Create(ctx context.Context, chatID int64, userID int64, senderUsername string, text string) (dom.Message, error)
// 	EditMessage(ctx context.Context, messageID int64, newText string) error
// 	DeleteMessage(ctx context.Context, messageID int64) error
// 	CheckMessageExists(ctx context.Context, messageID int64) (bool, error)
// 	ListByChat(ctx context.Context, chatID int64, limit, lastMessage int) ([]dom.Message, error)
// }

type MessageRepository interface {
	SaveMessage(ctx context.Context, msg interface{}) (string, error)
	EditMessage(ctx context.Context, senderID int64, chatID int64, msgID string, newText string) (int64, error)
	DeleteMessage(ctx context.Context, senderID, chatID int64, msgID []string) (int64, error)
	GetMessages(ctx context.Context, chatID int64, anchorTime time.Time, anchorID string, limit int64) ([]dom.Message, error)
}

type KafkaProducer interface {
	SendMessageCreated(ctx context.Context, event events.EventMessageCreated) error
}

type MessageService struct {
	Chat   ChatInterface
	Msg    MessageRepository
	Kafka  KafkaProducer
	Logger *slog.Logger
}

func NewMessageService(chat ChatInterface, msg MessageRepository, kafka KafkaProducer, logger *slog.Logger) *MessageService {
	return &MessageService{
		Chat:   chat,
		Msg:    msg,
		Kafka:  kafka,
		Logger: logger,
	}
}

func (m *MessageService) SendMessage(ctx context.Context, chatID int64, userID int64, senderUsername string, text string) error {
	isMember, err := m.Chat.CheckIsMemberOfChat(ctx, chatID, userID)
	if err != nil {
		return customerrors.ErrDatabase
	}
	if !isMember {
		return customerrors.ErrUserNotMemberOfChat
	}

	msg := dom.Message{
		ChatID:         chatID,
		SenderID:       userID,
		SenderUsername: senderUsername,
		Text:           text,
		CreatedAt:      time.Now(),
	}

	mongoID, err := m.Msg.SaveMessage(ctx, msg)
	if err != nil {
		return customerrors.ErrDatabase
	}

	event := events.EventMessageCreated{
		MessageID: mongoID,
		ChatID:    chatID,
		SenderID:  userID,
		CreatedAt: msg.CreatedAt,
	}

	if err := m.Kafka.SendMessageCreated(ctx, event); err != nil {
		m.Logger.Warn("failed to publish event", "error", err)
	}
	return nil
}

func (m *MessageService) DeleteMessage(ctx context.Context, senderID int64, chatID int64, msgID []string) error {

	if senderID <= 0 || chatID <= 0 || len(msgID) <= 0 {
		return customerrors.ErrInvalidInput
	}

	isMember, err := m.Chat.CheckIsMemberOfChat(ctx, chatID, senderID)
	if err != nil {
		return fmt.Errorf("failed to check if user is member of chat: %w", customerrors.ErrDatabase)
	}
	if !isMember {
		return customerrors.ErrUserNotMemberOfChat
	}

	deletedCount, err := m.Msg.DeleteMessage(ctx, senderID, chatID, msgID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	if deletedCount == 0 {
		return customerrors.ErrMessageDoesNotExists
	}
	return nil

}

func (m *MessageService) EditMessage(ctx context.Context, senderID int64, chatID int64, msgID string, newText string) error {

	if senderID <= 0 || chatID <= 0 || msgID == "" || newText == "" {
		return customerrors.ErrInvalidInput
	}

	isMember, err := m.Chat.CheckIsMemberOfChat(ctx, chatID, senderID)
	if err != nil {
		return customerrors.ErrDatabase
	}
	if !isMember {
		return customerrors.ErrUserNotMemberOfChat
	}

	updatedCount, err := m.Msg.EditMessage(ctx, senderID, chatID, msgID, newText)
	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}
	if updatedCount == 0 {
		return customerrors.ErrMessageDoesNotExists
	}
	return nil
}

func (m *MessageService) GetMessages(ctx context.Context, userID, chatID int64, anchorTime time.Time, anchorID string, limit int64) ([]dom.Message, error) {

	if userID <= 0 || chatID <= 0 || limit <= 0 {
		return nil, customerrors.ErrInvalidInput
	}

	isMember, err := m.Chat.CheckIsMemberOfChat(ctx, chatID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if user is member of chat: %w", customerrors.ErrDatabase)
	}
	if !isMember {
		return nil, customerrors.ErrUserNotMemberOfChat
	}
	return m.Msg.GetMessages(ctx, chatID, anchorTime, anchorID, limit)
}
