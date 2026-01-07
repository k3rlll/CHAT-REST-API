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

//go:generate mockgen -source=message_service.go -destination=mock/message_mocks.go -package=mock
type MessageInterface interface {
	Create(ctx context.Context, chatID int64, userID int64, senderUsername string, text string) (dom.Message, error)
	EditMessage(ctx context.Context, messageID int64, newText string) error
	DeleteMessage(ctx context.Context, messageID int64) error
	CheckMessageExists(ctx context.Context, messageID int64) (bool, error)
	ListByChat(ctx context.Context, chatID int64, limit, lastMessage int) ([]dom.Message, error)
}

type MongoRepository interface {
	Save(ctx context.Context, msg interface{}) (string, error)
}

type KafkaProducer interface {
	SendMessageCreated(ctx context.Context, event events.EventMessageCreated) error
}

type MessageService struct {
	Chat    ChatInterface
	Message MessageInterface
	Mongo   MongoRepository
	Kafka   KafkaProducer
	Logger  *slog.Logger
}

func NewMessageService(chat ChatInterface, message MessageInterface, mongo MongoRepository, kafka KafkaProducer, logger *slog.Logger) *MessageService {
	return &MessageService{
		Chat:    chat,
		Message: message,
		Mongo:   mongo,
		Kafka:   kafka,
		Logger:  logger,
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

	mongoID, err := m.Mongo.Save(ctx, msg)
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
		fmt.Errorf("failed to publish event: %w", err)
	}
	return nil
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
