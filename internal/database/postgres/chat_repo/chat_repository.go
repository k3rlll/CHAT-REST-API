package database

import (
	"context"
	"log/slog"
	domChat "main/internal/domain/chat"
	domMessage "main/internal/domain/message"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewChatRepository(pool *pgxpool.Pool, logger *slog.Logger) *ChatRepository {
	return &ChatRepository{
		pool:   pool,
		logger: logger,
	}
}

func (c *ChatRepository) CheckIsMemberOfChat(ctx context.Context, chatID int64, userID int64) (bool, error) {
	isMember := false
	err := c.pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM chat_members WHERE chat_id=$1 AND user_id=$2)", chatID, userID).Scan(&isMember)
	if err != nil {
		c.logger.Error("failed to check if user is a member of the chat", err)
		return false, err
	}
	return isMember, nil

}

func (c *ChatRepository) CreateChat(ctx context.Context,
	title string,
	isPrivate bool,
	membersID []int64) (int64, error) {

	var chatId int64
	var username string

	if len(membersID) ==1 {
		_, err:= c.pool.Exec(ctx,
			"select username from users where user_id=$1", membersID[0])
		if err != nil {
			c.logger.Error("failed to get username by userID", err.Error())
			return 0, err
		}
	}

	if len(membersID) == 2 {
		err := c.pool.QueryRow(ctx,
			"SELECT username FROM users WHERE user_id=$1", membersID[1]).Scan(&username)
		if err != nil {
			c.logger.Error("failed to get nickname by username", err.Error())
			return 0, err
		}
	}

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		c.logger.Error("failed to begin transaction", err.Error())
		return 0, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		"INSERT INTO chats (title, is_private) VALUES ($1, $2) RETURNING id", title, isPrivate).Scan(&chatId)
	if err != nil {
		c.logger.Error("failed to create a chat", err.Error())
		return 0, err
	}

	for _, userID := range membersID {
		_, err := tx.Exec(ctx,
			"INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)", chatId, userID)
		if err != nil {
			c.logger.Error("failed to add users to chat", err.Error())
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.logger.Error("failed to commit transaction", err.Error())
		return 0, err
	}

	return chatId, nil

}

func (c *ChatRepository) DeleteChat(ctx context.Context, chatID int64) error {
	_, err := c.pool.Exec(ctx,
		"DELETE FROM chats WHERE id=$1", chatID)
	return err
}

func (c *ChatRepository) CheckIfChatExists(ctx context.Context, chatID int64) (bool, error) {
	exists := false
	err := c.pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM chats WHERE id=$1)", chatID).Scan(&exists)
	return exists, err
}

func (c *ChatRepository) ListOfChats(ctx context.Context, userID int64) ([]domChat.Chat, error) {

	var chats []domChat.Chat
	query := "SELECT title FROM chats where id IN (SELECT chat_id FROM chat_members WHERE user_id=$1)"

	rows, err := c.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var chat domChat.Chat
		if err := rows.Scan(&chat.Title); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return chats, nil

}

func (c *ChatRepository) GetChatDetails(ctx context.Context, chatID int64) (domChat.Chat, error) {

	var chat domChat.Chat
	query := "SELECT id, title, is_private, created_at, members, members_usernames, members_count FROM chats WHERE id=$1"
	err := c.pool.QueryRow(ctx, query, chatID).Scan(&chat.Id, &chat.Title, &chat.IsPrivate, &chat.CreatedAt, &chat.Members, &chat.MembersUsernames, &chat.MembersCount)
	if err != nil {
		c.logger.Error("failed to get chat details", err.Error())
		return domChat.Chat{}, err
	}
	return domChat.Chat{
		Id:               chat.Id,
		Title:            chat.Title,
		IsPrivate:        chat.IsPrivate,
		CreatedAt:        chat.CreatedAt,
		Members:          chat.Members,
		MembersUsernames: chat.MembersUsernames,
		MembersCount:     chat.MembersCount,
	}, nil
}

func (c *ChatRepository) OpenChat(ctx context.Context, chatID int64, userID int64) ([]domMessage.Message, error) {

	rows, err := c.pool.Query(ctx,
		"SELECT messages.id, messages.chat_id, messages.sender, messages.text, messages.created_at "+
			"FROM messages "+
			"JOIN chat_members ON messages.chat_id = chat_members.chat_id "+
			"WHERE chat_members.user_id = $1 AND messages.chat_id = $2 "+
			"ORDER BY messages.created_at DESC", userID, chatID)
	if err != nil {
		return nil, err
	}

	messages := []domMessage.Message{}
	for rows.Next() {
		var message domMessage.Message
		if err := rows.Scan(&message.Id,
			&message.ChatID,
			&message.SenderID,
			&message.SenderUsername,
			&message.Text,
			&message.CreatedAt); err != nil {
			c.logger.Error("failed to scan message", err.Error())
			return nil, err
		}
		messages = append(messages, message)
	}

	defer rows.Close()

	return messages, nil
}

func (c *ChatRepository) AddMembers(ctx context.Context, chatID int64, members []int64) error {
	for _, userID := range members {
		_, err := c.pool.Exec(ctx,
			"INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)", chatID, userID)
		if err != nil {
			c.logger.Error("failed to add member to chat", err.Error())
			return err
		}
	}
	return nil
}

func (c *ChatRepository) UserInChat(ctx context.Context, chatID int64, userID int64) (bool, error) {
	isMember := false
	err := c.pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM chat_members WHERE chat_id=$1 AND user_id=$2)", chatID, userID).Scan(&isMember)
	return isMember, err
}

var _ domChat.ChatRepository = (*ChatRepository)(nil)
