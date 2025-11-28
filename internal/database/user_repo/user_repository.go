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
	email string,
	passwordHash string) (dom.User, error) {
	var userRes dom.User

	if CheckEmailExists(ctx, r.pool, email) {
		r.logger.Error("email already exists", customerrors.ErrEmailAlreadyExists.Error())
		return dom.User{}, customerrors.ErrEmailAlreadyExists
	}

	tag, err := r.pool.Exec(ctx,
		"INSERT INTO users (nickname, email, password_hash) VALUES ($1, $2, $3)",
		username, email, passwordHash)
	if err != nil {
		r.logger.Error("failed to insert new user", err.Error())
		return dom.User{}, err
	}

	if tag.RowsAffected() == 0 {
		r.logger.Error("no rows affected when inserting new user")
		return dom.User{}, err
	}

	_ = r.pool.QueryRow(ctx,
		"SELECT ID FROM users WHERE email=$1", email).Scan(userRes.ID)

	return userRes, nil
}

func (r *UserRepository) SearchUser(ctx context.Context, q string) ([]dom.User, error) {

	const sqlq = `
SELECT id, nickname
FROM users
WHERE
      email = $1
   OR CAST(id AS TEXT) = $1
   OR nickname = $1
   OR nickname ILIKE '%' || $1 || '%'
ORDER BY
   (CAST(id AS TEXT) = $1) DESC,
   (nickname = $1) DESC,
   id;`
	rows, err := r.pool.Query(ctx, sqlq, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []dom.User
	for rows.Next() {
		var u dom.User
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
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

var _ dom.UserRepository = (*UserRepository)(nil)
