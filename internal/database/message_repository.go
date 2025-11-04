package database

import (
	"context"
	"log/slog"
	"main/internal/pkg/customerrors"
	"main/internal/service"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Message struct {
	Id        int
	Text      string
	CreatedAt time.Time
	ChatID    int
	UserID    int
}

type messageRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewMessageRepository(pool *pgxpool.Pool, logger *slog.Logger) *messageRepository {
	return &messageRepository{
		pool:   pool,
		logger: logger,
	}
}

type MessageRepository interface {
}

func (m *messageRepository) DeleteMessage(ctx context.Context, id int) error {

	exists, err := service.CheckMessageExists(ctx, m.pool, id, m.logger)
	if err != nil {
		m.logger.Error(err.Error())
		return err
	}

	if !exists {
		return customerrors.ErrMessageDoesNotExists
	} else {
		_, err = m.pool.Exec(ctx, "DELETE FROM messages WHERE id=$1", id)
		if err != nil {
			m.logger.Error("failed to delete a message", err.Error())
			return err
		}
	}

	return nil
}

func (m *messageRepository) Create(ctx context.Context, chatID int, userID int, text string) (int, error) {
	var messageID int

	err := m.pool.QueryRow(ctx,
		"INSERT INTO messages (chat_id, user_id, text) VALUES ($1,$2,$3) RETURNING id", chatID, userID, text).Scan(&messageID)
	if err != nil {
		m.logger.Error("failed to create message", err.Error())
		return 0, err
	}

	return messageID, nil
}
