package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	dom "main/internal/domain/chat"
	domMessage "main/internal/domain/message"
	"main/internal/pkg/customerrors"
)

type ChatService struct {
	User   UserInterface
	Chat   ChatInterface
	Logger *slog.Logger
}

type ChatInterface interface {
	GetChatDetails(ctx context.Context, chatID int64) (dom.Chat, error)
	ListOfChats(ctx context.Context, userID int64) ([]dom.Chat, error)
	CheckIfChatExists(ctx context.Context, chatID int64) (bool, error)
	DeleteChat(ctx context.Context, chatID int64) error
	CreateChat(ctx context.Context, title string, isPrivate bool, members []int64) (int64, error)
	CheckIsMemberOfChat(ctx context.Context, chatID int64, userID int64) (bool, error)
	OpenChat(ctx context.Context, chatID int64, userID int64) ([]domMessage.Message, error)
	UserInChat(ctx context.Context, chatID int64, userID int64) (bool, error)
	AddMembers(ctx context.Context, chatID int64, members []int64) error
}

type UserInterface interface {
	CheckUserExists(ctx context.Context, userID int64) bool
}

func NewChatService(user UserInterface, chat ChatInterface, logger *slog.Logger) *ChatService {
	return &ChatService{
		User:   user,
		Chat:   chat,
		Logger: logger,
	}
}
func (c *ChatService) CreateChat(ctx context.Context,
	isPrivate bool,
	title string,
	members []int64) (dom.Chat, error) {

	if len(members) == 0 {
		return dom.Chat{}, fmt.Errorf("chat service: amount of members cannot be less than 0: %w", customerrors.ErrInvalidInput)
	}

	if title == "" {
		return dom.Chat{}, fmt.Errorf("chat service:chat title cannot be empty: %w", customerrors.ErrInvalidInput)
	}

	chat_id, err := c.Chat.CreateChat(ctx, title, isPrivate, members)
	if err != nil {
		return dom.Chat{}, err
	}

	chat := dom.Chat{
		Id:        chat_id,
		Title:     title,
		IsPrivate: isPrivate,
	}
	return chat, nil
}

func (c *ChatService) DeleteChat(ctx context.Context, chatID int64) error {
	c.Logger.Info("DeleteChat called", slog.Int64("chatID", chatID))

	exists, err := c.Chat.CheckIfChatExists(ctx, chatID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("chat does not exist")
	}

	err = c.Chat.DeleteChat(ctx, chatID)
	if err != nil {
		return err
	}
	return nil
}

func (c *ChatService) ListOfChats(ctx context.Context, userID int64) ([]dom.Chat, error) {

	return c.Chat.ListOfChats(ctx, userID)

}

func (c *ChatService) GetChatDetails(ctx context.Context, chatID int64, userID int64) (dom.Chat, error) {
	c.Logger.Info("GetChatDetails called", slog.Int64("chatID", chatID), slog.Int64("userID", userID))

	isMember, err := c.Chat.CheckIsMemberOfChat(ctx, chatID, userID)
	if err != nil {
		return dom.Chat{}, customerrors.ErrFailedToCheck
	}
	if !isMember {
		return dom.Chat{}, customerrors.ErrUserNotMemberOfChat
	}
	chat, err := c.Chat.GetChatDetails(ctx, chatID)
	if err != nil {
		return dom.Chat{}, err
	}
	return chat, nil
}

func (c *ChatService) OpenChat(ctx context.Context,
	chatID int64,
	userID int64) (dom.Chat,
	[]domMessage.Message,
	error) {

	isMember, err := c.Chat.CheckIsMemberOfChat(ctx, chatID, userID)
	if err != nil {
		return dom.Chat{}, nil, customerrors.ErrFailedToCheck
	}
	if !isMember {
		return dom.Chat{}, nil, customerrors.ErrUserNotMemberOfChat
	}

	details, err := c.Chat.GetChatDetails(ctx, chatID)
	if err != nil {
		return dom.Chat{}, nil, err
	}
	for i, member := range details.MembersID {
		details.MembersID[i] = member
	}
	details.MembersCount = len(details.MembersID)
	details.MembersID = nil

	messages, err := c.Chat.OpenChat(ctx, chatID, userID)
	if err != nil {
		return dom.Chat{}, nil, err
	}

	return details, messages, nil
}

func (c *ChatService) AddMembers(ctx context.Context, chatID int64, userID int64, members []int64) error {
	if !c.User.CheckUserExists(ctx, userID) {
		return customerrors.ErrUserNotFound
	}

	inChat, err := c.Chat.UserInChat(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if inChat {
		return customerrors.ErrUserAlreadyInChat
	}

	isMember, err := c.Chat.CheckIsMemberOfChat(ctx, chatID, userID)
	if err != nil {
		return customerrors.ErrFailedToCheck
	}
	if !isMember {
		return customerrors.ErrUserNotMemberOfChat
	}

	err = c.Chat.AddMembers(ctx, chatID, members)
	if err != nil {
		return err
	}

	return nil
}
