package user

import (
	"context"
	"log/slog"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	"main/internal/pkg/utils"
	"strings"
	"time"
)

//go:generate mockgen -source=user_service.go -destination=mock/user_mock.go -package=mock
type UserInterface interface {
	RegisterUser(ctx context.Context, username, email, passwordHash string) (dom.User, error)
	SearchUser(ctx context.Context, query string) ([]dom.User, error)
}

type UserService struct {
	Repo     UserInterface
	Logger   *slog.Logger
	Timeout  time.Duration
	MaxLimit int64
}

func NewUserService(repo UserInterface, logger *slog.Logger) *UserService {
	if logger == nil {
		logger = slog.Default()
	}
	return &UserService{
		Repo:     repo,
		Logger:   logger,
		Timeout:  3 * time.Hour,
		MaxLimit: 100,
	}
}

func (s *UserService) RegisterUser(ctx context.Context, username, email, password string) (dom.User, error) {

	if !utils.ValidatePassword(password) {
		return dom.User{}, customerrors.ErrInvalidInput
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return dom.User{}, err
	}

	res, err := s.Repo.RegisterUser(ctx, username, email, passwordHash)
	if err != nil {
		return dom.User{}, customerrors.ErrDatabase
	}

	res = dom.User{
		ID:       res.ID,
		Username: res.Username,
		Email:    res.Email,
	}
	return res, nil

}

func (s *UserService) SearchUser(ctx context.Context, message string) ([]dom.User, error) {
	q := strings.TrimSpace(message)
	if q == "" {
		return []dom.User{}, customerrors.ErrInvalidInput
	}

	users, err := s.Repo.SearchUser(ctx, q)
	if err != nil {
		return []dom.User{}, err
	}

	return users, nil

}
