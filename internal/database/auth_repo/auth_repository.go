package auth_repo

import (
	"context"
	"log/slog"
	"main/internal/pkg/customerrors"
	"main/internal/pkg/jwt"
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
        INSERT INTO refresh_tokens (user_id, refresh_token, expires_at)
        VALUES ($1, $2, $3)`,
		userID, refreshToken, expiresAt)

	return err
}

func (t *TokenRepository) Login(ctx context.Context, token *jwt.TokenPair, userID int64, password string) (*jwt.TokenPair, error) {
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

func (t *TokenRepository) Logout(ctx context.Context, userID int64, token jwt.TokenPair) error {

	_, err := t.pool.Exec(ctx,
		"UPDATE refresh_tokens SET is_revorked=true WHERE user_id=$1 AND token=$2", userID, token.RefreshToken)
	if err != nil {
		t.logger.Error("failed to logout user", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (t *TokenRepository) LogoutAll(ctx context.Context, userID int64) error {
	_, err := t.pool.Exec(ctx,
		"UPDATE refresh_tokens SET is_revorked=true WHERE user_id=$1", userID)
	if err != nil {
		t.logger.Error("failed to logout all user sessions", slog.String("error", err.Error()))
		return err
	}
	return nil
}
