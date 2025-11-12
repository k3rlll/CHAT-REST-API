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

func (s *AuthService) CheckIfBlocked(ctx context.Context, userID int64) (bool, error) {

}

func (s *AuthService) LoginUser(ctx context.Context, userID int64, password string) (*jwt.TokenPair, error) {
	tokenPair, err := jwt.GenerateJWT(userID)
	if err != nil {
		s.Logger.Error("failed to generate JWT tokens", slog.String("error", err.Error()))
		return nil, err
	}

	return s.Repo.Login(ctx, tokenPair, userID, password)
}

// TODO: попробовать подключить Redis для хранения токенов, также для дальнейшей работы с токенами
func (s *AuthService) LogoutUser(ctx context.Context, userID int64, token jwt.TokenPair) error {

	return s.Repo.Logout(ctx, userID, token)
}

func (s *AuthService) LogoutAllSessions(ctx context.Context, userID int64) error {
	return s.Repo.LogoutAll(ctx, userID)
}
