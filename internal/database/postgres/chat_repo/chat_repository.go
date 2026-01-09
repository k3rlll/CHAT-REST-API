package chat_repo

import (
	"context"
	"fmt"
	"log/slog"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	"time"

	"github.com/jackc/pgx/v5"
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

func (c *ChatRepository) CreateChat(ctx context.Context,
	title string,
	isPrivate bool,
	membersID []int64) (int64, error) {

	var chatId int64

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", customerrors.ErrDatabase)
	}
	defer tx.Rollback(ctx)
	if title == "" {
		for _, userID := range membersID {
			var username string
			err := tx.QueryRow(ctx,
				"SELECT username FROM users WHERE id=$1", userID).Scan(&username)
			if err != nil {
				return 0, fmt.Errorf("chat repository: failed to select username: %w", customerrors.ErrDatabase)
			}
			if userID != membersID[len(membersID)-1] {
				title += username + ", "
			} else {
				title += username
			}
			title += username
		}

	}
	err = tx.QueryRow(ctx,
		"INSERT INTO chats (title, is_private) VALUES ($1, $2) RETURNING id", title, isPrivate).Scan(&chatId)
	if err != nil {
		return 0, fmt.Errorf("failed to insert chat: %w", customerrors.ErrDatabase)
	}

	err = c.addMembersTX(ctx, tx, chatId, membersID)
	if err != nil {
		return 0, fmt.Errorf("failed to add members to chat: %w", customerrors.ErrDatabase)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", customerrors.ErrDatabase)
	}

	return chatId, nil

}
func (c *ChatRepository) addMembersTX(ctx context.Context, tx pgx.Tx, chatID int64, members []int64) error {
	for _, userID := range members {
		_, err := tx.Exec(ctx,
			"INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)", chatID, userID)
		if err != nil {
			return fmt.Errorf("repository: failed to insert chat member: %w", err)
		}
	}
	return nil

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

func (c *ChatRepository) ListOfChats(ctx context.Context, userID int64) ([]dom.Chat, error) {

	var chats []dom.Chat
	query := "SELECT title FROM chats where id IN (SELECT chat_id FROM chat_members WHERE user_id=$1) ORDER BY last_message_at DESC"

	rows, err := c.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to select titles of chats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var chat dom.Chat
		if err := rows.Scan(&chat.Title); err != nil {
			return nil, fmt.Errorf("failed to scan rows: %w", err)
		}
		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return chats, nil

}

func (c *ChatRepository) GetChatDetails(ctx context.Context, chatID int64) (dom.Chat, error) {

	var chat dom.Chat
	query := "SELECT id, title, is_private, created_at, members, members_usernames, members_count FROM chats WHERE id=$1"
	err := c.pool.QueryRow(ctx, query, chatID).Scan(
		&chat.ID,
		&chat.Title,
		&chat.IsPrivate,
		&chat.CreatedAt,
		&chat.MembersID,
		&chat.MembersUsernames,
		&chat.MembersCount)
	if err != nil {
		return dom.Chat{}, fmt.Errorf("failed to select chat details: %w", err)
	}
	return dom.Chat{
		ID:               chat.ID,
		Title:            chat.Title,
		IsPrivate:        chat.IsPrivate,
		CreatedAt:        chat.CreatedAt,
		MembersID:        chat.MembersID,
		MembersUsernames: chat.MembersUsernames,
		MembersCount:     chat.MembersCount,
	}, nil
}

// func (c *ChatRepository) OpenChat(ctx context.Context, chatID int64, userID int64) ([]dom.Message, error) {

// 	rows, err := c.pool.Query(ctx,
// 		"SELECT messages.id, messages.chat_id, messages.sender, messages.text, messages.created_at "+
// 			"FROM messages "+
// 			"JOIN chat_members ON messages.chat_id = chat_members.chat_id "+
// 			"WHERE chat_members.user_id = $1 AND messages.chat_id = $2 "+
// 			"ORDER BY messages.created_at DESC", userID, chatID)
// 	if err != nil {
// 		return nil, fmt.Errorf("repository: failed to select messages: %w", err)
// 	}

// 	messages := []dom.Message{}
// 	for rows.Next() {
// 		var message dom.Message
// 		if err := rows.Scan(
// 			&message.Id,
// 			&message.ChatID,
// 			&message.SenderID,
// 			&message.SenderUsername,
// 			&message.Text,
// 			&message.CreatedAt); err != nil {
// 			return nil, fmt.Errorf("repository: failed to scan message: %w", err)
// 		}
// 		messages = append(messages, message)
// 	}

// 	defer rows.Close()

// 	return messages, nil
// }

func (c *ChatRepository) AddMembers(ctx context.Context, chatID int64, members []int64) error {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, userID := range members {
		_, err := tx.Exec(ctx,
			"INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)", chatID, userID)
		if err != nil {
			return fmt.Errorf("repository: failed to insert chat member: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("repository: failed to commit transaction: %w", err)
	}
	return nil
}

func (c *ChatRepository) CheckIsMemberOfChat(ctx context.Context, chatID int64, userID int64) (bool, error) {
	isMember := false
	query := "SELECT EXISTS (SELECT 1 FROM chat_members WHERE chat_id=$1 AND user_id=$2)"
	err := c.pool.QueryRow(ctx,
		query, chatID, userID).Scan(&isMember)
	return isMember, err
}

func (c *ChatRepository) RemoveMember(ctx context.Context, chatID, userID int64) error {
	_, err := c.pool.Exec(ctx,
		"DELETE FROM chat_members WHERE chat_id=$1 AND user_id=$2", chatID, userID)
	return err
}

func (c *ChatRepository) UpdateChatLastMessage(ctx context.Context, chatID int64, at time.Time) error {
	_, err := c.pool.Exec(ctx,
		"UPDATE chats SET last_message_at=$1 WHERE id=$2", at, chatID)
	return err
}
