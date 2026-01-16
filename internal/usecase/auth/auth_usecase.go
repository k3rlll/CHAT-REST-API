package auth

import (
	"context"
	"fmt"
	"time"

	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	"main/internal/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
)

//go:generate mockgen -source=auth_usecase.go -destination=./mock/auth_usecase_mock.go -package=mock
type AuthRepository interface {
	GetCredentialsByUsername(ctx context.Context, username string) (dom.User, error)
	SaveRefreshToken(ctx context.Context, refreshToken dom.RefreshToken) error
	GetRefreshToken(ctx context.Context, userID int64) (string, error)
	DeleteRefreshToken(ctx context.Context, userID int64) error
}

type TokenManager interface {
	NewAccessToken(userID int64, ttl time.Duration) (string, error)
	NewRefreshToken() (string, error)
	Parse(accessToken string) (*jwt.TokenClaims, error)
}

type TokenBlacklister interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

type AuthService struct {
	repo      AuthRepository
	tokenMgr  TokenManager
	blacklist TokenBlacklister
	tokenTTL  time.Duration
}

func NewAuthService(repo AuthRepository, tokenMgr TokenManager, blacklist TokenBlacklister, tokenTTL time.Duration) *AuthService {
	return &AuthService{
		repo:      repo,
		tokenMgr:  tokenMgr,
		blacklist: blacklist,
		tokenTTL:  tokenTTL,
	}
}

// Обновленный метод LoginUser
func (s *AuthService) LoginUser(ctx context.Context, username, password string) (string, dom.RefreshToken, error) {

	user, err := s.repo.GetCredentialsByUsername(ctx, username)
	if err != nil {
		return "", dom.RefreshToken{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {

		return "", dom.RefreshToken{}, customerrors.ErrInvalidInput
	}

	refreshTokenString, err := s.tokenMgr.NewRefreshToken()
	if err != nil {
		return "", dom.RefreshToken{}, fmt.Errorf("service: generate refresh: %w", err)
	}

	accessTokenString, err := s.tokenMgr.NewAccessToken(user.ID, s.tokenTTL)
	if err != nil {
		return "", dom.RefreshToken{}, fmt.Errorf("service: generate access: %w", err)
	}

	refreshToken := dom.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenString,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour * 15),
	}

	if err := s.repo.SaveRefreshToken(ctx, refreshToken); err != nil {
		return "", dom.RefreshToken{}, err
	}

	return accessTokenString, refreshToken, nil
}

func (s *AuthService) LogoutUser(ctx context.Context, accessToken, refreshToken string) error {
	claims, err := s.tokenMgr.Parse(accessToken)
	if err != nil {
		return fmt.Errorf("invalid access token: %w", err)
	}

	expirationTime := time.Unix(claims.Exp, 0)
	ttl := time.Until(expirationTime)

	if ttl > 0 {
		if err := s.blacklist.Set(ctx, accessToken, "blacklisted", ttl); err != nil {
			return fmt.Errorf("redis blacklist error: %w", err)
		}
	}

	if err := s.repo.DeleteRefreshToken(ctx, claims.UserID); err != nil {
		return fmt.Errorf("db error: %w", err)
	}

	return nil
}
