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

func NewUserRepository(pool *pgxpool.Pool, logger *slog.Logger) *UserRepository {
	return &UserRepository{
		pool:   pool,
		logger: logger,
	}
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

	if tag.RowsAffected() == 0 {
		r.logger.Error("no rows affected when inserting new user")
		return User{}, err
	}
	//TODO: подумать над ошибкой которую возвращать

	//TODO: подумай надо что то выводить или нет
	return userRes, nil
}

func (r *UserRepository) LogIn(ctx context.Context, password string) error {
	var passwordHash string

	_ = r.pool.QueryRow(ctx,
		"SELECT password_hash FROM users WHERE password_hash=$1", passwordHash)

	if !service.CheckPasswordHash(password, passwordHash) {
		r.logger.Error("invalid password", customerrors.ErrInvalidNicknameOrPassword.Error())
		return customerrors.ErrInvalidNicknameOrPassword
	}

	return nil
}
