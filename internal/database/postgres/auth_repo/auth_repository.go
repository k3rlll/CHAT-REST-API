package auth_repo

import (
	"context"
	"log/slog"
	dom "main/internal/domain/auth"
	domUser "main/internal/domain/user"
	"main/internal/pkg/customerrors"

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

func (t *AuthRepository) SaveRefreshToken(ctx context.Context,
	userID int64,
	refreshToken string) error {

	_, err := t.pool.Exec(ctx, `
        INSERT INTO refresh_tokens (user_id, refresh_token)
        VALUES ($1, $2)`,
		userID, refreshToken)

	return err
}

func (t *AuthRepository) GetByEmail(ctx context.Context, email string) (domUser.User, error) {
	var user domUser.User

	err := t.pool.QueryRow(ctx,
		"SELECT id, username, password_hash FROM users WHERE email=$1", email).
		Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		return domUser.User{}, err
	}
	return user, nil
}

func (t *AuthRepository) GetPasswordHash(ctx context.Context,
	RefreshToken string,
	userID int64,
	password string) (domUser.User, error) {
	var passwordHash string

	err := t.pool.QueryRow(ctx,
		"SELECT password_hash FROM users WHERE id=$1", userID).Scan(&passwordHash)
	if err != nil {
		return domUser.User{}, customerrors.ErrInvalidNicknameOrPassword
	}
	var user domUser.User
	user.ID = userID
	user.Password = passwordHash

	return user, nil
}

func (t *AuthRepository) DeleteRefreshToken(ctx context.Context, userID int64) error {
	_, err := t.pool.Exec(ctx,
		"DELETE FROM refresh_tokens WHERE user_id=$1", userID)
	return err
}

var _ dom.AuthInterface = (*AuthRepository)(nil)
