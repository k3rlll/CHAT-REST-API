package message_repo

import (
	"context"
	"log/slog"
	dom "main/internal/domain/message"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewMessageRepository(pool *pgxpool.Pool, logger *slog.Logger) *MessageRepository {
	return &MessageRepository{
		pool:   pool,
		logger: logger,
	}
}

// type MessageRepository interface {
// }

func (m *MessageRepository) CheckMessageExists(ctx context.Context, id int64) (bool, error) {
	exists := false

	err := m.pool.QueryRow(ctx,
		"SELECT EXISTS (text FROM messages WHERE id=$1)", id).Scan(&exists)
	if err != nil {
		m.logger.Error("failed to check if message exists", err.Error())
		return false, err
	}
	return exists, nil
}

func (m *MessageRepository) DeleteMessage(ctx context.Context, id int64) error {
	_, err := m.pool.Exec(ctx, "DELETE FROM messages WHERE id=$1", id)
	if err != nil {
		m.logger.Error("failed to delete a message", err.Error())
		return err
	}

	return nil
}

func (m *MessageRepository) Create(ctx context.Context, chatID int64, userID int64, text string) (dom.Message, error) {
	var messageID int64

	err := m.pool.QueryRow(ctx,
		"INSERT INTO messages (chat_id, user_id, text) VALUES ($1,$2,$3) RETURNING id", chatID, userID, text).Scan(&messageID)
	if err != nil {
		m.logger.Error("failed to create message", err.Error())
		return dom.Message{}, err
	}

	var res = dom.Message{
		Id:        messageID,
		Text:      text,
		CreatedAt: time.Now(),
		ChatID:    chatID,
		SenderID:  userID,
	}

	return res, nil
}

func (m *MessageRepository) EditMessage(ctx context.Context, messageID int64, newText string) error {
	_, err := m.pool.Exec(ctx,
		"UPDATE messages SET text=$1 WHERE id=$2", newText, messageID)
	if err != nil {
		m.logger.Error("failed to edit message", err.Error())
		return err
	}
	return nil
}

// ListByChat отправляет список сообщений для указанного чата с учетом лимита и смещения.
func (m *MessageRepository) ListByChat(ctx context.Context, chatID int64) ([]dom.Message, error) {
	rows, err := m.pool.Query(ctx,
		`SELECT id, chat_id, user_id, text, created_at
		   FROM messages
		  WHERE chat_id = $1
		  ORDER BY created_at DESC`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []dom.Message
	for rows.Next() {
		var m dom.Message
		if err := rows.Scan(&m.Id, &m.ChatID, &m.SenderID, &m.Text, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

var _ dom.MessageRepository = (*MessageRepository)(nil)
