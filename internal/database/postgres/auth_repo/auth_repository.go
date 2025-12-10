package auth_repo

import (
	"context"
	"log/slog"
	dom "main/internal/domain/auth"
	domUser "main/internal/domain/user"
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

func (t *TokenRepository) SaveRefreshToken(ctx context.Context,
	userID int64,
	refreshToken string) error {

	_, err := t.pool.Exec(ctx, `
        INSERT INTO refresh_tokens (user_id, refresh_token)
        VALUES ($1, $2)`,
		userID, refreshToken)

	return err
}

func (t *TokenRepository) GetByEmail(ctx context.Context, email string) (domUser.User, error) {
	var user domUser.User

	err := t.pool.QueryRow(ctx,
		"SELECT id, username, password_hash FROM users WHERE email=$1", email).
		Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		return domUser.User{}, err
	}
	return user, nil
}

func (t *TokenRepository) Login(ctx context.Context,
	RefreshToken string,
	userID int64,
	password string) (domUser.User, error) {
	var passwordHash string

	err := t.pool.QueryRow(ctx,
		"SELECT password_hash FROM users WHERE id=$1", userID).Scan(&passwordHash)
	if err != nil {
		return domUser.User{}, customerrors.ErrInvalidNicknameOrPassword
	}


	return domUser.User{}, nil
}

var _ dom.TokenRepository = (*TokenRepository)(nil)
