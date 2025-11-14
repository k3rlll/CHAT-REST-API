package auth

import (
	"context"
	"log/slog"
	db "main/internal/domain/auth"
	"main/internal/pkg/jwt"

	"github.com/go-redis/redis"
)

type AuthService struct {
	Redis  *redis.Client
	Repo   db.TokenRepository
	Logger *slog.Logger
}

func NewAuthService(repo db.TokenRepository, logger *slog.Logger) *AuthService {
	return &AuthService{

		Repo:   repo,
		Logger: logger,
	}
}

// func (s *AuthService) ValidateRefreshToken(ctx context.Context, userID int64, password string) (*jwt.TokenPair, error) {}

func (s *AuthService) LoginUser(ctx context.Context, userID int64, password string) (*db.TokenPair, error) {

	tokenPair, err := jwt.GenerateJWT(userID)
	if err != nil {
		s.Logger.Error("failed to generate JWT tokens", slog.String("error", err.Error()))
		return nil, err
	}

	return s.Repo.Login(ctx, tokenPair, userID, password)
}
