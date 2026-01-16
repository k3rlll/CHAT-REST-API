package auth_repo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"

	"github.com/jackc/pgx/v5"
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

func (r *AuthRepository) GetCredentialsByUsername(ctx context.Context, username string) (dom.User, error) {
	var user dom.User

	query := "SELECT id, password_hash FROM users WHERE username=$1"

	err := r.pool.QueryRow(ctx, query, username).Scan(&user.ID, &user.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dom.User{}, customerrors.ErrUserNotFound
		}
		return dom.User{}, fmt.Errorf("repo: get credentials: %w", err)
	}

	user.Username = username
	return user, nil
}

func (r *AuthRepository) SaveRefreshToken(ctx context.Context, rt dom.RefreshToken) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, refresh_token, created_at, expires_at)
		VALUES ($1, $2, $3, $4)`,
		rt.UserID, rt.Token, rt.CreatedAt, rt.ExpiresAt)

	if err != nil {
		return fmt.Errorf("repo: save refresh token: %w", customerrors.ErrDatabase)
	}
	return nil
}

func (r *AuthRepository) GetRefreshToken(ctx context.Context, userID int64) (string, error) {
	var storedToken string
	var expiry time.Time

	err := r.pool.QueryRow(ctx,
		"SELECT refresh_token, expires_at FROM refresh_tokens WHERE user_id=$1", userID).
		Scan(&storedToken, &expiry)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", customerrors.ErrRefreshTokenNotFound
		}
		return "", fmt.Errorf("repo: get refresh token: %w", err)
	}

	if time.Now().After(expiry) {
		_ = r.DeleteRefreshToken(ctx, userID)
		return "", customerrors.ErrRefreshTokenExpired
	}

	return storedToken, nil
}

func (r *AuthRepository) DeleteRefreshToken(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM refresh_tokens WHERE user_id=$1", userID)
	if err != nil {
		return fmt.Errorf("repo: delete refresh token: %w", err)
	}
	return nil
}
