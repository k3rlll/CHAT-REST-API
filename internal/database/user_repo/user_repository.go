package user_repository

import (
	"context"
	"log/slog"
	dom "main/internal/domain/user"
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
	nickname string,
	email string,
	passwordHash string) (dom.User, error) {
	var userRes dom.User

	if r.CheckUsernameExists(ctx, nickname) {
		r.logger.Error("username already exists", customerrors.ErrUsernameAlreadyExists)
		return dom.User{}, customerrors.ErrUsernameAlreadyExists
	}

	if CheckEmailExists(ctx, r.pool, email) {
		r.logger.Error("email already exists", customerrors.ErrEmailAlreadyExists)
		return dom.User{}, customerrors.ErrEmailAlreadyExists
	}

	tag, err := r.pool.Exec(ctx,
		"INSERT INTO users (nickname, email, password_hash, username) VALUES ($1, $2, $3, $4);",
		nickname, email, passwordHash, username)
	if err != nil {
		r.logger.Error("failed to insert new user", err.Error())
		return dom.User{}, err
	}

	if tag.RowsAffected() == 0 {
		r.logger.Error("no rows affected when inserting new user")
		return dom.User{}, err
	}
	userRes = dom.User{
		Nickname: nickname,
		Email:    email,
		Username: username,
	}

	return userRes, nil
}

func (r *UserRepository) SearchUser(ctx context.Context, q string) ([]dom.User, error) {
	var users []dom.User
	rows, err := r.pool.Query(ctx,
		"SELECT username FROM users WHERE nickname ILIKE '%' || $1 || '%' OR username ILIKE '%' || $1 || '%';", q)
	if err != nil {
		r.logger.Error("error occurred during user search", slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user dom.User
		err := rows.Scan(&user.Username)
		if err != nil {
			r.logger.Error("error occurred during scanning user search result", slog.String("error", err.Error()))
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
		r.logger.Error("error occurred during user existence check", slog.String("error", err.Error()))
		return false
	}
	return exists
}

var _ dom.UserRepository = (*UserRepository)(nil)
