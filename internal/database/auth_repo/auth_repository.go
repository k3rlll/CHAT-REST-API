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

func (t *TokenRepository) Login(ctx context.Context, token *jwt.TokenPair, userID int64, password string) (*jwt.TokenPair, error) {
	var passwordHash string

	_ = t.pool.QueryRow(ctx,
		"SELECT password_hash FROM users WHERE password_hash=$1", passwordHash)

	if !utils.CheckPasswordHash(password, passwordHash) {
		t.logger.Error("failed to login: invalid password")
		return nil, customerrors.ErrInvalidNicknameOrPassword
	}

	_ = t.pool.QueryRow(ctx,
		"INSERT INTO users (refresh_token) VALUES ($1)", token.RefreshToken)

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
