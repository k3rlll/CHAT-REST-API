package user

import (
	"context"
	"log/slog"
	db "main/internal/database/user_repo"
	dom "main/internal/domain/user"
	"main/internal/pkg/customerrors"
	"main/internal/pkg/utils"
	"strings"
	"time"
)

type SearchUsersParams struct {
	Q      string
	Limit  int
	Offset int
}

type service struct {
	Repo     *db.UserRepository
	Logger   *slog.Logger
	Timeout  time.Duration
	MaxLimit int
}

func NewUserService(repo *db.UserRepository, logger *slog.Logger) *service {
	if logger == nil {
		logger = slog.Default()
	}
	return &service{
		Repo:     repo,
		Logger:   logger,
		Timeout:  3 * time.Second,
		MaxLimit: 100,
	}
}

func (s *service) RegisterUser(ctx context.Context, username, email, password string) (dom.User, error) {

	if !utils.ValidatePassword(password) {
		s.Logger.Error("password does not meet complexity requirements")
		return dom.User{}, customerrors.ErrInvalidPassword
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		s.Logger.Error("failed to hash password", err.Error())
	}

	res, err := s.Repo.RegisterUser(ctx, username, email, passwordHash)
	if err != nil {
		s.Logger.Error("failed to register user", err.Error())
		return dom.User{}, err
	}
	return res, nil

}

func (s *service) SearchUser(ctx context.Context, p SearchUsersParams) ([]dom.User, error) {
	q := strings.TrimSpace(p.Q)
	if q == "" {
		return []dom.User{}, customerrors.ErrEmptyQuery
	}

	limit := p.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > s.MaxLimit {
		limit = s.MaxLimit
	}
	offset := p.Offset
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	users, err := s.Repo.SearchUser(ctx, q, limit, offset)
	if err != nil {
		s.Logger.Error("failed to search users", err.Error())
		return []dom.User{}, err
	}

	return users, nil

}
