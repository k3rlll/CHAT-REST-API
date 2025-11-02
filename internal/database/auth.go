package database

import (
	"context"
	"log/slog"
	"main/internal/pkg/customerrors"
	"main/internal/pkg/jwt"
	"main/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepository interface {
	Login(ctx context.Context, userID int64, password string) (*jwt.TokenPair, error)
	Logout(ctx context.Context, userID int, token jwt.TokenPair) error
	LogoutAll(ctx context.Context, userID int64) (int64, error)
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

func (t *tokenRepository) Login(ctx context.Context, userID int, password string) (*jwt.TokenPair, error) {
	var passwordHash string

	token, err := jwt.GenerateJWT(int64(userID))
	if err != nil {
		t.logger.Error("invalid")
	}

	_ = t.pool.QueryRow(ctx,
		"SELECT password_hash FROM users WHERE password_hash=$1", passwordHash)

	if !service.CheckPasswordHash(password, passwordHash) {
		t.logger.Error("invalid password", customerrors.ErrInvalidNicknameOrPassword.Error())
		return nil, customerrors.ErrInvalidNicknameOrPassword
	}

	_ = t.pool.QueryRow(ctx,
		"INSERT INTO users (refresh_token) VALUES ($1)", token.RefreshToken)

	return token, nil
}

func (t *tokenRepository) Logout(ctx context.Context, userID int, token jwt.TokenPair) error {

	_, err := t.pool.Exec(ctx,
		"UPDATE refresh_tokens SET is_revorked=true WHERE user_id=$1 AND token=$2", userID, token.RefreshToken)
	if err != nil {
		t.logger.Error("failed to logout user", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (t *tokenRepository) LogoutAll(ctx context.Context, userID int) error {
	_, err := t.pool.Exec(ctx,
		"UPDATE refresh_tokens SET is_revorked=true WHERE user_id=$1", userID)
	if err != nil {
		t.logger.Error("failed to logout all user sessions", slog.String("error", err.Error()))
		return err
	}
	return nil
}
