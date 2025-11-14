package auth_repo

import (
	"context"
	"log/slog"
	dom "main/internal/domain/auth"
	"main/internal/pkg/customerrors"
	"main/internal/pkg/utils"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewTokenRepository(pool *pgxpool.Pool, logger *slog.Logger) *TokenRepository {
	return &TokenRepository{
		pool:   pool,
		logger: logger,
	}
}

func (t *TokenRepository) SaveRefreshToken(ctx context.Context, userID int64, refreshToken string) error {

	_, err := t.pool.Exec(ctx, `
        INSERT INTO refresh_tokens (user_id, refresh_token)
        VALUES ($1, $2)`,
		userID, refreshToken)

	return err
}

func (t *TokenRepository) Login(ctx context.Context, token *dom.TokenPair, userID int64, password string) (*dom.TokenPair, error) {
	var passwordHash string

	err := t.pool.QueryRow(ctx,
		"SELECT password_hash FROM users WHERE id=$1", userID).Scan(&passwordHash)
	if err != nil {
		t.logger.Error("failed to get user password hash", slog.String("error", err.Error()))
		return nil, customerrors.ErrInvalidNicknameOrPassword
	}

	if !utils.CheckPasswordHash(password, passwordHash) {
		t.logger.Error("failed to login: invalid password")
		return nil, customerrors.ErrInvalidNicknameOrPassword
	}

	if err := t.SaveRefreshToken(ctx, userID, token.RefreshToken); err != nil {
		t.logger.Error("failed to save refresh token", slog.String("error", err.Error()))
		return nil, err
	}

	return token, nil
}

