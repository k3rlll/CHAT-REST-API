package chat

import (
	"context"
	"errors"
	"log/slog"
	dom "main/internal/domain/chat"
)

type ChatService struct {
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
