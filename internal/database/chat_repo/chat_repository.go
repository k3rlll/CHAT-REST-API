package database

import (
	"context"
	"log/slog"

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

func (c *ChatRepository) CheckIsMemberOfChat(ctx context.Context, chatID int, userID int, logger *slog.Logger) (bool, error) {
	isMember := false
	err := c.pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM chat_members WHERE chat_id=$1 AND user_id=$2)", chatID, userID).Scan(&isMember)
	if err != nil {
		return false, err
	}
	return isMember, nil

}

func (c *ChatRepository) CreateChat(ctx context.Context, name string, isPrivate bool, userIDs []int) (string, int, error) {

	var chatId int
	username := ""

	if len(userIDs) == 2 {
		userID2 := userIDs[1]
		err := c.pool.QueryRow(ctx,
			"SELECT username FROM users WHERE id=$1", userID2).Scan(&username)
		if err != nil {
			c.logger.Error("failed to get username by id", err.Error())
			return "", 0, err
		}
	}

	if username != "" {
		name = username
	}

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		c.logger.Error("failed to begin transaction", err.Error())
		return "", 0, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		"INSERT INTO chats (name, is_private) VALUES ($1, $2) RETURNING id", name, isPrivate).Scan(&chatId)
	if err != nil {
		c.logger.Error("failed to create a chat", err.Error())
		return "", 0, err
	}

	for _, userID := range userIDs {
		_, err := tx.Exec(ctx,
			"INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)", chatId, userID)
		if err != nil {
			c.logger.Error("failed to add users to chat", err.Error())
			return "", 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.logger.Error("failed to commit transaction", err.Error())
		return "", 0, err
	}

	return "", chatId, nil

}
