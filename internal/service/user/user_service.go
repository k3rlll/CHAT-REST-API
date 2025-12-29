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

//go:generate mockery --name=UserInterface --output=./mocks --with-expecter --case=snake
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
		s.Logger.Error("password does not meet complexity requirements")
		return dom.User{}, customerrors.ErrInvalidInput
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		s.Logger.Error("failed to hash password", err.Error())
		return dom.User{}, err
	}

	res, err := s.Repo.RegisterUser(ctx, username, email, passwordHash)
	if err != nil {
		s.Logger.Error("failed to register user", err.Error())
		return dom.User{}, err
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
		s.Logger.Error("empty search query", slog.String("query", q))
		return []dom.User{}, customerrors.ErrInvalidInput
	}

	users, err := s.Repo.SearchUser(ctx, q)
	if err != nil {
		s.Logger.Info("service", "UserService.SearchUser")
		return []dom.User{}, err
	}
	s.Logger.Info("users search results retrieved successfully", slog.Int("count", len(users)),
		slog.String("service", "UserService.SearchUser"))

	return users, nil

}
