package auth

import (
	"context"
	"log/slog"
	db "main/internal/domain/auth"
	"main/internal/pkg/jwt"
	"strconv"

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
	redisKey := "block_user" + strconv.FormatInt(userID, 10)
	result, err := s.Redis.Get(redisKey).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return result == "blocked", nil
}

func (s *AuthService) LoginUser(ctx context.Context, userID int64, password string) (*jwt.TokenPair, error) {
	blocked, err := s.CheckIfBlocked(ctx, userID)
	if err != nil {
		s.Logger.Info("failed to check if user is blocked", err)
		return nil, err
	}
	if blocked {

	}
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
