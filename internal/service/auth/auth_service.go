package auth

import (
	"context"
	"log/slog"
	db "main/internal/domain/auth"
	"time"

	customerrors "main/internal/pkg/customerrors"
	jwt "main/internal/pkg/jwt"
	"main/internal/pkg/utils"
)

type AuthService struct {
	jwt    Token
	Repo   db.AuthInterface
	Logger *slog.Logger
}

type Token interface {
	NewAccessToken(userID int64, ttl time.Duration) (string, error)
	Parse(accessToken string) (int64, error)
	NewRefreshToken() (string, error)
}

func NewTokenService() Token {
	return &jwt.Claims{}
}

func NewAuthService(repo db.AuthInterface, logger *slog.Logger, jwt Token) *AuthService {
	return &AuthService{
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

func (s *AuthService) LogoutUser(ctx context.Context, userID int64) error {

	err := s.Repo.DeleteRefreshToken(ctx, userID)
	if err != nil {
		return err
	}

	return nil
}
