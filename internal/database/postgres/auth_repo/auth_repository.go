package auth_repo

import (
	"context"
	"fmt"
	"log/slog"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewAuthRepository(pool *pgxpool.Pool, logger *slog.Logger) *AuthRepository {
	return &AuthRepository{
		pool:   pool,
		logger: logger,
	}
}

func (t *AuthRepository) SaveRefreshToken(ctx context.Context, refreshToken dom.RefreshToken) error {

	_, err := t.pool.Exec(ctx, `
        INSERT INTO refresh_tokens (user_id, refresh_token, created_at, expires_at)
        VALUES ($1, $2, $3, $4)`,
		refreshToken.UserID, refreshToken.Token, refreshToken.CreatedAt, refreshToken.ExpiresAt)

	if err != nil {
		return err
	}
	return nil
}

func (t *AuthRepository) GetRefreshToken(ctx context.Context, userId int64) (string, error) {
	var storedToken string
	var expiry time.Time
	err := t.pool.QueryRow(ctx,
		"SELECT refresh_token, expires_at from refresh_tokens WHERE user_id=$1", userId).
		Scan(&storedToken, &expiry)
	if err != nil {
		return "", fmt.Errorf("get refresh token:%w", customerrors.ErrRefreshTokenNotFound)
	}
	if storedToken == "" {
		return "", fmt.Errorf("repository:refresh token not found: %w", customerrors.ErrRefreshTokenNotFound)
	}
	if time.Now().After(expiry) {
		if err := t.DeleteRefreshToken(ctx, userId); err != nil {
			return "", fmt.Errorf("repository:delete expired refresh token: %w", err)
		}
		return "", fmt.Errorf("repository:refresh token expired: %w", customerrors.ErrRefreshTokenExpired)
	}

	return storedToken, nil
}

func (t *AuthRepository) GetByEmail(ctx context.Context, email string) (dom.User, error) {
	var user dom.User

	err := t.pool.QueryRow(ctx,
		"SELECT id, username, password_hash FROM users WHERE email=$1", email).
		Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		return dom.User{}, err
	}
	return user, nil
}

func (t *AuthRepository) GetPasswordHash(ctx context.Context,
	userID int64,
	password string) (dom.User, error) {
	var passwordHash string

	err := t.pool.QueryRow(ctx,
		"SELECT password_hash FROM users WHERE id=$1", userID).Scan(&passwordHash)
	if err != nil {
		return dom.User{}, customerrors.ErrInvalidInput
	}
	var user dom.User
	user.ID = userID
	user.Password = passwordHash

	return user, nil
}

func (t *AuthRepository) DeleteRefreshToken(ctx context.Context, userID int64) error {
	_, err := t.pool.Exec(ctx,
		"DELETE FROM refresh_tokens WHERE user_id=$1", userID)
	return err
}
