package service

import (
	"context"
	"log/slog"
	db "main/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type messageService struct {
	Repo   db.MessageRepository
	Logger *slog.Logger
}

type MessageService interface {
}

func CheckMessageExists(ctx context.Context, pool *pgxpool.Pool, id int, logger *slog.Logger) (bool, error) {
	exists := false

	err := pool.QueryRow(ctx,
		"SELECT EXISTS (text FROM messages WHERE id=$1)", id).Scan(&exists)
	if err != nil {
		logger.Error("failed to check if message exists", err.Error())
		return false, err
	}
	return exists, nil
}
