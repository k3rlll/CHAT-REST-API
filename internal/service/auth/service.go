package auth

import (
	"context"
	"log/slog"
	db "main/internal/database/auth_repo"
	"main/internal/pkg/jwt"
)

type service struct {
	Repo   *db.TokenRepository
	Logger *slog.Logger
}

func NewAuthService(repo *db.TokenRepository, logger *slog.Logger) *service {
	return &service{
		Repo:   repo,
		Logger: logger,
	}
}

func (s *service) LoginUser(ctx context.Context, userID int64, password string) (*jwt.TokenPair, error) {
	tokenPair, err := jwt.GenerateJWT(userID)
	if err != nil {
		s.Logger.Error("failed to generate JWT tokens", slog.String("error", err.Error()))
		return nil, err
	}

	return s.Repo.Login(ctx, tokenPair, userID, password)
}
