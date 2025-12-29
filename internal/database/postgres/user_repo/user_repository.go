package user_repo

import (
	"context"
	"log/slog"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewUserRepository(pool *pgxpool.Pool, logger *slog.Logger) *UserRepository {
	return &UserRepository{
		pool:   pool,
		logger: logger,
	}
}

func CheckEmailExists(ctx context.Context, pool *pgxpool.Pool, email string) bool {
	var exists bool
	_ = pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", email).Scan(&exists)

	return exists
}

func (r *UserRepository) RegisterUser(
	ctx context.Context,
	username string,
	email string,
	passwordHash string) (dom.User, error) {
	var userRes dom.User

	if r.CheckUsernameExists(ctx, username) {
		return dom.User{}, customerrors.ErrUsernameAlreadyExists
	}

	if CheckEmailExists(ctx, r.pool, email) {
		return dom.User{}, customerrors.ErrEmailAlreadyExists
	}

	tag, err := r.pool.Exec(ctx,
		"INSERT INTO users (email, password_hash, username) VALUES ($1, $2, $3);",
		email, passwordHash, username)
	if err != nil {
		return dom.User{}, err
	}

	if tag.RowsAffected() == 0 {
		return dom.User{}, err
	}
	userRes = dom.User{
		Username: username,
		Email:    email,
	}

	return userRes, nil
}

func (r *UserRepository) SearchUser(ctx context.Context, q string) ([]dom.User, error) {
	var users []dom.User
	rows, err := r.pool.Query(ctx,
		"SELECT username FROM users WHERE username ILIKE '%' || $1 || '%';", q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user dom.User
		err := rows.Scan(&user.Username)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepository) CheckUsernameExists(ctx context.Context, username string) bool {
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM users WHERE username=$1)", username).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func (r *UserRepository) CheckUserExists(ctx context.Context, userID int64) bool {
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM users WHERE id=$1)", userID).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func (r *UserRepository) ChangeUsername(ctx context.Context, username string) (dom.User, error) {
	var user dom.User
	if !r.CheckUsernameExists(ctx, username) {
		_, err := r.pool.Exec(ctx,
			"UPDATE users SET username=$1", username)
		if err != nil {
			return dom.User{}, err
		}
	}
	return user, nil
}
