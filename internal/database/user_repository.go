package database

import (
	"context"
	"log/slog"
	"main/internal/pkg/customerrors"
	"main/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UserRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

type UserRepositoryInterface interface {
	RegisterUser(ctx context.Context, username, email, password string) (User, error)
	SearchUser(ctx context.Context, q string, limit, offset int) ([]User, error)
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
	password string) (User, error) {
	var userRes User

	passwordHash := service.HashPassword(password)

	tag, err := r.pool.Exec(ctx,
		"INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3)",
		username, email, passwordHash)
	if err != nil {
		r.logger.Error("failed to insert new user", err.Error())
		return User{}, err
	}

	if CheckEmailExists(ctx, r.pool, email) {
		r.logger.Error("email already exists")
		return User{}, customerrors.ErrEmailAlreadyExists
	}

	if !service.ValidatePassword(password) {
		r.logger.Error("password does not meet complexity requirements")
		return User{}, customerrors.ErrInvalidPassword
	}

	if tag.RowsAffected() == 0 {
		r.logger.Error("no rows affected when inserting new user")
		return User{}, err
	}

	//TODO: подумай надо что то выводить или нет
	return userRes, nil
}

func (r *UserRepository) SearchUser(ctx context.Context, q string, limit, offset int) ([]User, error) {
	const sqlq = `
SELECT id, email, nickname
FROM users
WHERE
      email = $1
   OR CAST(id AS TEXT) = $1
   OR nickname = $1
   OR email ILIKE '%' || $1 || '%'
   OR nickname ILIKE '%' || $1 || '%'
ORDER BY
   (email = $1) DESC,
   (CAST(id AS TEXT) = $1) DESC,
   (nickname = $1) DESC,
   id
LIMIT $2 OFFSET $3;
`
	rows, err := r.pool.Query(ctx, sqlq, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Username); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
