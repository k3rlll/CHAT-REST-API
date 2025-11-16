package chat

import (
	"context"
	"errors"
	"log/slog"
	dom "main/internal/domain/chat"
	domMessage "main/internal/domain/message"
	domUser "main/internal/domain/user"
	"main/internal/pkg/customerrors"
	"strings"
)

type ChatService struct {
	User   domUser.UserRepository
	Chat   dom.ChatRepository
	Logger *slog.Logger
}

func NewChatService(chat dom.ChatRepository, logger *slog.Logger) *ChatService {
	return &ChatService{
		Chat:   chat,
		Logger: logger,
	}
}
func (c *ChatService) CreateChat(ctx context.Context, isPrivate bool, title string, members []int) (dom.Chat, error) {
	if title == "" {
		c.Logger.Info("title is empty", errors.New("title cannot be empty"))
		return dom.Chat{}, errors.New("title cannot be empty")
	}
	if len(members) < 2 {
		c.Logger.Info("not enough members to create a chat", errors.New("a chat must have at least two members"))
		return dom.Chat{}, errors.New("a chat must have at least two members")
	}

	chat_id, err := c.Chat.CreateChat(ctx, title, isPrivate, members)
	if err != nil {
		c.Logger.Error("failed to create chat", err.Error())
		return dom.Chat{}, err
	}

	chat := dom.Chat{
		Id:        chat_id,
		Title:     title,
		IsPrivate: isPrivate,
	}
	return chat, nil
}

func (c *ChatService) DeleteChat(ctx context.Context, chatID int) error {

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

func (c *ChatService) ListOfChats(ctx context.Context) ([]dom.Chat, error) {

	return c.Chat.ListOfChats(ctx)

}

func (c *ChatService) GetChatDetails(ctx context.Context, chatID int, userID int) (dom.Chat, error) {

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

func (c *ChatService) OpenChat(ctx context.Context, chatID int, userID int) (dom.Chat, []domMessage.Message, error) {

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
	for i, member := range details.Members {
		details.Members[i] = strings.TrimSpace(member)

	}
	details.MembersCount = len(details.Members)
	details.Members = nil

	messages, err := c.Chat.OpenChat(ctx, chatID, userID)
	if err != nil {
		c.Logger.Error("failed to open chat", err.Error())
		return dom.Chat{}, nil, err
	}

	return details, messages, nil
}

func (c *ChatService) AddMembers(ctx context.Context, chatID int, UserID int, members []int) error {
	if c.User.CheckUserExists(ctx, UserID) {
		c.Logger.Info("user does not exist", nil)
		return customerrors.ErrUserDoesNotExist
	}

	inChat, err := c.Chat.UserInChat(ctx, chatID, UserID)
	if err != nil {
		c.Logger.Error("failed to check if user is in chat", err.Error())
		return err
	}
	if inChat {
		c.Logger.Info("user is already in the chat", nil)
		return customerrors.ErrUserAlreadyInChat
	}

	isMember, err := c.Chat.CheckIsMemberOfChat(ctx, chatID, UserID)
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
