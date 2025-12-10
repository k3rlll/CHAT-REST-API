package auth

import (
	"context"
	"log/slog"
	db "main/internal/domain/auth"
	jwt "main/internal/pkg/jwt"
	"time"

	customerrors "main/internal/pkg/customerrors"
	"main/internal/pkg/utils"

	"github.com/go-redis/redis"
)

type AuthService struct {
	jwt    jwt.Token
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

	AccessToken, err = s.jwt.NewAccessToken(userID, time.Minute*15)
	if err != nil {
		return "", "", err
	}

	RefreshToken, err = s.jwt.NewRefreshToken()
	if err != nil {
		return "", "", err
	}


	if !utils.CheckPasswordHash(password, user.Password) {
		return "", "", customerrors.ErrInvalidNicknameOrPassword
	}


	return AccessToken, RefreshToken, nil
}
