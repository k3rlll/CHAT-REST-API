package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	dom "main/internal/domain/chat"
	domMessage "main/internal/domain/message"
	domUser "main/internal/domain/user"
	"main/internal/pkg/customerrors"
)

type ChatService struct {
	User   domUser.UserInterface
	Chat   dom.ChatInterface
	Logger *slog.Logger
}

type ChatInterface interface {
	GetChatDetails(ctx context.Context, chatID int64) (Chat, error)
	ListOfChats(ctx context.Context, userID int64) ([]Chat, error)
	CheckIfChatExists(ctx context.Context, chatID int64) (bool, error)
	DeleteChat(ctx context.Context, chatID int64) error
	CreateChat(ctx context.Context, title string, isPrivate bool, members []int64) (int64, error)
	CheckIsMemberOfChat(ctx context.Context, chatID int64, userID int64) (bool, error)
	OpenChat(ctx context.Context, chatID int64, userID int64) ([]domMessage.Message, error)
	AddMembers(ctx context.Context, chatID int64, members []int64) error
	UserInChat(ctx context.Context, chatID int64, userID int64) (bool, error)
}

func NewChatService(user domUser.UserInterface, chat dom.ChatInterface, logger *slog.Logger) *ChatService {
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

	exists, err := c.Chat.CheckIfChatExists(ctx, chatID)
	if err != nil {
		c.Logger.Error("failed to check if chat exists", err.Error())
		return err
	}
	if !exists {
		c.Logger.Info("chat does not exist", nil)
		return errors.New("chat does not exist")
	}

	err = c.Chat.DeleteChat(ctx, chatID)
	if err != nil {
		c.Logger.Error("failed to delete chat", err.Error())
		return err
	}
	return nil
}

func (c *ChatService) ListOfChats(ctx context.Context, userID int64) ([]dom.Chat, error) {

	return c.Chat.ListOfChats(ctx, userID)

}

func (c *ChatService) GetChatDetails(ctx context.Context, chatID int64, userID int64) (dom.Chat, error) {

	isMember, err := c.Chat.CheckIsMemberOfChat(ctx, chatID, userID)
	if err != nil {
		c.Logger.Error("failed to check membership", err.Error())
		return dom.Chat{}, customerrors.ErrFailedToCheck
	}
	if !isMember {
		c.Logger.Info("user is not a member of the chat", nil)
		return dom.Chat{}, customerrors.ErrUserNotMemberOfChat
	}
	chat, err := c.Chat.GetChatDetails(ctx, chatID)
	if err != nil {
		c.Logger.Error("failed to get chat details", err.Error())
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
		c.Logger.Error("failed to check membership", err.Error())
		return dom.Chat{}, nil, customerrors.ErrFailedToCheck
	}
	if !isMember {
		c.Logger.Info("user is not a member of the chat", nil)
		return dom.Chat{}, nil, customerrors.ErrUserNotMemberOfChat
	}

	details, err := c.Chat.GetChatDetails(ctx, chatID)
	if err != nil {
		c.Logger.Error("failed to get chat details", err.Error())
		return dom.Chat{}, nil, err
	}
	for i, member := range details.MembersID {
		details.MembersID[i] = member
	}
	details.MembersCount = len(details.MembersID)
	details.MembersID = nil

	messages, err := c.Chat.OpenChat(ctx, chatID, userID)
	if err != nil {
		c.Logger.Error("failed to open chat", err.Error())
		return dom.Chat{}, nil, err
	}

	return details, messages, nil
}

func (c *ChatService) AddMembers(ctx context.Context, chatID int64, userID int64, members []int64) error {
	if !c.User.CheckUserExists(ctx, userID) {
		c.Logger.Info("user does not exist", nil)
		return customerrors.ErrUserNotFound
	}

	inChat, err := c.Chat.UserInChat(ctx, chatID, userID)
	if err != nil {
		c.Logger.Error("failed to check if user is in chat", err.Error())
		return err
	}
	if inChat {
		c.Logger.Info("user is already in the chat", nil)
		return customerrors.ErrUserAlreadyInChat
	}

	isMember, err := c.Chat.CheckIsMemberOfChat(ctx, chatID, userID)
	if err != nil {
		c.Logger.Error("failed to check membership", err.Error())
		return customerrors.ErrFailedToCheck
	}
	if !isMember {
		c.Logger.Info("user is not a member of the chat", nil)
		return customerrors.ErrUserNotMemberOfChat
	}

	err = c.Chat.AddMembers(ctx, chatID, members)
	if err != nil {
		c.Logger.Error("failed to add members to chat", err.Error())
		return err
	}

	return nil
}
