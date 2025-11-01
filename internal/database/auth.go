package database

import (
	"context"
	"log/slog"
	"main/internal/pkg/customerrors"
	"main/internal/service"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepository interface {
	InsertRefresh(ctx context.Context, userID int64, sessionID, tokenHash string, issuedAt, expiresAt time.Time, ua, ip string) error
	GetBySession(ctx context.Context, sessionID string) (userID int64, tokenHash string, expiresAt time.Time, err error)
	DeleteBySession(ctx context.Context, sessionID string) error
	DeleteAllByUser(ctx context.Context, userID int64) (int64, error)
}

type tokenRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewTokenRepository(pool *pgxpool.Pool, logger *slog.Logger) *tokenRepository {
	return &tokenRepository{
		pool:   pool,
		logger: logger,
	}
}

func (t *tokenRepository) Login(ctx context.Context, password string) error {
	var passwordHash string

	_ = t.pool.QueryRow(ctx,
		"SELECT password_hash FROM users WHERE password_hash=$1", passwordHash)

	if !service.CheckPasswordHash(password, passwordHash) {
		t.logger.Error("invalid password", customerrors.ErrInvalidNicknameOrPassword.Error())
		return customerrors.ErrInvalidNicknameOrPassword
	}

	return nil
}

func (t *tokenRepository) Logout() error {
	return nil
}
