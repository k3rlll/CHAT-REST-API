// package message_repo

// import (
// 	"context"
// 	"fmt"
// 	dom "main/internal/domain/entity"
// 	"time"

// 	"github.com/jackc/pgx/v5/pgxpool"
// )

// type MessageRepository struct {
// 	pool *pgxpool.Pool
// }

// func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
// 	return &MessageRepository{
// 		pool: pool,
// 	}
// }

// func (m *MessageRepository) CheckMessageExists(ctx context.Context, id int64) (bool, error) {
// 	exists := false

// 	err := m.pool.QueryRow(ctx,
// 		"SELECT EXISTS (SELECT 1 FROM messages WHERE id=$1)", id).Scan(&exists)
// 	if err != nil {
// 		return false, fmt.Errorf("repository:failed to check if message exists: %w", err)
// 	}
// 	return exists, nil
// }

// func (m *MessageRepository) DeleteMessage(ctx context.Context, id int64) error {
// 	_, err := m.pool.Exec(ctx, "DELETE FROM messages WHERE id=$1", id)
// 	if err != nil {
// 		return fmt.Errorf("repository:failed to delete message: %w", err)
// 	}

// 	return nil
// }

// func (m *MessageRepository) Create(
// 	ctx context.Context,
// 	chatID int64, userID int64,
// 	senderUsername string, text string) (dom.Message, error) {
// 	var messageID int64
// 	var createdAt time.Time
// 	query := "INSERT INTO messages (chat_id, sender_id, sender_username, text) VALUES ($1,$2,$3,$4) RETURNING id, created_at"

// 	err := m.pool.QueryRow(ctx,
// 		query, chatID, userID, senderUsername, text).Scan(&messageID, &createdAt)
// 	if err != nil {

// 		return dom.Message{}, fmt.Errorf("repository:failed to create message: %w", err)
// 	}

// 	var res = dom.Message{
// 		Id:             messageID,
// 		Text:           text,
// 		ChatID:         chatID,
// 		SenderID:       userID,
// 		SenderUsername: senderUsername,
// 		CreatedAt:      createdAt,
// 	}

// 	return res, nil
// }

// func (m *MessageRepository) EditMessage(ctx context.Context, messageID int64, newText string) error {
// 	_, err := m.pool.Exec(ctx,
// 		"UPDATE messages SET text=$1 WHERE id=$2", newText, messageID)
// 	if err != nil {
// 		return fmt.Errorf("repository:failed to edit message: %w", err)
// 	}
// 	return nil
// }

// func (m *MessageRepository) ListByChat(ctx context.Context, chatID int64, limit, lastMessage int64) ([]dom.Message, error) {

// 	query := `
// 		SELECT id, chat_id, sender_id, sender_username, text, created_at
// 		FROM messages
// 		WHERE chat_id = $1 AND id < $2
// 		ORDER BY created_at DESC
// 		LIMIT $3`

// 	rows, err := m.pool.Query(ctx, query, chatID, lastMessage, limit)
// 	if err != nil {
// 		return nil, fmt.Errorf("repository: failed to list messages: %w", err)
// 	}
// 	defer rows.Close()

// 	out := make([]dom.Message, 0, limit)

// 	for rows.Next() {
// 		var m dom.Message
// 		if err := rows.Scan(&m.Id, &m.ChatID, &m.SenderID, &m.SenderUsername, &m.Text, &m.CreatedAt); err != nil {
// 			return nil, fmt.Errorf("scan error: %w", err)
// 		}
// 		out = append(out, m)
// 	}

// 	if err := rows.Err(); err != nil {
// 		return nil, fmt.Errorf("rows iteration error: %w", err)
// 	}

// 	return out, nil
// }
