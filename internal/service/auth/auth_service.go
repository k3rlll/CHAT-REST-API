package auth

import (
	"context"
	"log/slog"
	db "main/internal/domain/auth"
	"time"

	customerrors "main/internal/pkg/customerrors"
	jwt "main/internal/pkg/jwt"
	"main/internal/pkg/utils"

	"github.com/go-redis/redis/v8"
)

type AuthService struct {
	redis  *redis.Client
	jwt    Token
	Repo   db.AuthInterface
	Logger *slog.Logger
}

type Token interface {
	NewAccessToken(userID int64, ttl time.Duration) (string, error)
	NewRefreshToken() (string, error)
}

func NewTokenService() Token {
	return &jwt.Claims{}
}

func NewAuthService(repo db.AuthInterface, logger *slog.Logger, jwt Token, redis *redis.Client) *AuthService {
	return &AuthService{
		redis:  redis,
		jwt:    jwt,
		Repo:   repo,
		Logger: logger,
	}
}

func (s *AuthService) LoginUser(ctx context.Context,
	userID int64,
	password string) (
	AccessToken string,
	RefreshToken string,
	err error) {

	user, err := s.Repo.Login(ctx, RefreshToken, userID, password)
	if err != nil {
		return "", "", err
	}

	if !utils.CheckPasswordHash(password, user.Password) {
		return "", "", customerrors.ErrInvalidNicknameOrPassword
	}

	AccessToken, err = s.jwt.NewAccessToken(userID, time.Minute*15)
	if err != nil {
		return "", "", err
	}

	RefreshToken, err = s.jwt.NewRefreshToken()
	if err != nil {
		return "", "", err
	}

	err = s.Repo.SaveRefreshToken(ctx, userID, RefreshToken)
	if err != nil {
		return "", "", err
	}

	return AccessToken, RefreshToken, nil
}

func (s *AuthService) LogoutUser(ctx context.Context, userID int64, accessToken string) error {

	s.redis.Set(ctx, accessToken, "blacklist", time.Minute*15)

	err := s.Repo.DeleteRefreshToken(ctx, userID)
	if err != nil {
		return err
	}

	return nil
}
