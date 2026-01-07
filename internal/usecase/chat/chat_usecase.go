package chat

import (
	"context"
	"fmt"
	"log/slog"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
)

type ChatService struct {
	User   UserInterface
	Chat   ChatRepositoryInterface
	Logger *slog.Logger
}

//go:generate mockgen -source=chat_service.go -destination=mock/chat_mocks.go -package=mock
type ChatRepositoryInterface interface {
	GetChatDetails(ctx context.Context, chatID int64) (dom.Chat, error)
	ListOfChats(ctx context.Context, userID int64) ([]dom.Chat, error)
	CheckIfChatExists(ctx context.Context, chatID int64) (bool, error)
	DeleteChat(ctx context.Context, chatID int64) error
	CreateChat(ctx context.Context, title string, isPrivate bool, members []int64) (int64, error)
	CheckIsMemberOfChat(ctx context.Context, chatID int64, userID int64) (bool, error)
	OpenChat(ctx context.Context, chatID int64, userID int64) ([]dom.Message, error)
	AddMembers(ctx context.Context, chatID int64, members []int64) error
	RemoveMember(ctx context.Context, chatID int64, userID int64) error
}

type UserInterface interface {
	CheckUserExists(ctx context.Context, userID int64) bool
}

func NewChatService(user UserInterface, chat ChatRepositoryInterface, logger *slog.Logger) *ChatService {
	return &ChatService{
		User:   user,
		Chat:   chat,
		Logger: logger,
	}
}
func (c *ChatService) CreateChat(
	ctx context.Context,
	title string,
	isPrivate bool,
	members []int64) (dom.Chat, error) {

	if len(members) == 0 {
		return dom.Chat{}, fmt.Errorf("chat service: amount of members cannot be less than 0: %w", customerrors.ErrInvalidInput)
	}

	if title == "" {
		return dom.Chat{}, fmt.Errorf("chat service:chat title cannot be empty: %w", customerrors.ErrInvalidInput)
	}

	if len(title) > 20 {
		return dom.Chat{}, fmt.Errorf("chat service: chat title cannot be more than 20 characters: %w", customerrors.ErrInvalidInput)
	}

	chat_id, err := c.Chat.CreateChat(ctx, title, isPrivate, members)
	if err != nil {
		return dom.Chat{}, customerrors.ErrDatabase
	}

	chat := dom.Chat{
		Id:        chat_id,
		Title:     title,
		IsPrivate: isPrivate,
	}
	return chat, nil
}

func (c *ChatService) DeleteChat(ctx context.Context, chatID int64) error {
	if chatID <= 0 {
		return fmt.Errorf("chat service: invalid chatID: %w", customerrors.ErrInvalidInput)
	}

	exists, err := c.Chat.CheckIfChatExists(ctx, chatID)
	if err != nil {
		return err
	}
	if !exists {
		return customerrors.ErrNotFound
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
	[]dom.Message,
	error) {

	exists, err := c.Chat.CheckIfChatExists(ctx, chatID)
	if err != nil {
		return dom.Chat{}, nil, err
	}
	if !exists {
		return dom.Chat{}, nil, customerrors.ErrNotFound
	}

	if chatID <= 0 {
		return dom.Chat{}, nil, fmt.Errorf("chat service: invalid chatID: %w", customerrors.ErrInvalidInput)
	}

	isMember, err := c.Chat.CheckIsMemberOfChat(ctx, chatID, userID)
	if err != nil {
		return dom.Chat{}, nil, customerrors.ErrFailedToCheck
	}
	if !isMember {
		return dom.Chat{}, nil, customerrors.ErrUserNotMemberOfChat
	}

	details, err := c.Chat.GetChatDetails(ctx, chatID)
	if err != nil {
		return dom.Chat{}, nil, customerrors.ErrDatabase
	}

	details.MembersCount = len(details.MembersID)

	
	// messages, err := c.Chat.OpenChat(ctx, chatID, userID)
	// if err != nil {
	// 	return dom.Chat{}, nil, err
	// }

	// return details, messages, nil
}

func (c *ChatService) AddMembers(ctx context.Context, chatID int64, userID int64, members []int64) error {
	if !c.User.CheckUserExists(ctx, userID) {
		return customerrors.ErrUserNotFound
	}
	inChat, err := c.Chat.CheckIsMemberOfChat(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if !inChat {
		return customerrors.ErrUserNotMemberOfChat
	}

	for _, memberID := range members {
		if !c.User.CheckUserExists(ctx, memberID) {
			return customerrors.ErrUserNotFound
		}
		inChat, err := c.Chat.CheckIsMemberOfChat(ctx, chatID, memberID)
		if err != nil {
			return err
		}
		if inChat {
			return customerrors.ErrUserAlreadyInChat
		}
	}

	err = c.Chat.AddMembers(ctx, chatID, members)
	if err != nil {
		return err
	}

	return nil
}

func (c *ChatService) RemoveMember(ctx context.Context, chatID int64, userID int64) error {

	if chatID <= 0 || userID <= 0 {
		return fmt.Errorf("chat service: invalid chatID or userID: %w", customerrors.ErrInvalidInput)
	}

	isMember, err := c.Chat.CheckIsMemberOfChat(ctx, chatID, userID)
	if err != nil {
		return customerrors.ErrFailedToCheck
	}
	if !isMember {
		return customerrors.ErrUserNotMemberOfChat
	}
	err = c.Chat.RemoveMember(ctx, chatID, userID)
	if err != nil {
		return err
	}
	return nil
}
