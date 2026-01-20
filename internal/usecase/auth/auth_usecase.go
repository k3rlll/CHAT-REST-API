package auth

import (
	"context"
	"fmt"
	"time"

	dom "main/internal/domain/entity"
	
	"main/pkg/customerrors"
	"main/pkg/jwt"
	"main/pkg/utils"

	"golang.org/x/crypto/bcrypt"
)

//go:generate mockgen -source=auth_usecase.go -destination=./mock/auth_usecase_mock.go -package=mock
type AuthRepository interface {
	GetCredentialsByUsername(ctx context.Context, username string) (dom.User, error)
	SaveRefreshToken(ctx context.Context, refreshToken dom.RefreshToken) error
	GetRefreshToken(ctx context.Context, userID int64) (string, error)
	DeleteRefreshToken(ctx context.Context, userID int64) error
}
type UserRepository interface {
	RegisterUser(ctx context.Context, username, email, passwordHash string) (dom.User, error)
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
	repoAuth  AuthRepository
	repoUser  UserRepository
	tokenMgr  TokenManager
	blacklist TokenBlacklister
	tokenTTL  time.Duration
}

func NewAuthService(repoAuth AuthRepository, repoUser UserRepository, tokenMgr TokenManager, blacklist TokenBlacklister, tokenTTL time.Duration) *AuthService {
	return &AuthService{
		repoAuth:  repoAuth,
		repoUser:  repoUser,
		tokenMgr:  tokenMgr,
		blacklist: blacklist,
		tokenTTL:  tokenTTL,
	}
}

func (s *AuthService) LoginUser(ctx context.Context, username, password string) (accessToken string, err error) {

	user, err := s.repoAuth.GetCredentialsByUsername(ctx, username)
	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {

		return "", customerrors.ErrInvalidInput
	}

	refreshTokenString, err := s.tokenMgr.NewRefreshToken()
	if err != nil {
		return "", fmt.Errorf("service: generate refresh: %w", err)
	}

	accessTokenString, err := s.tokenMgr.NewAccessToken(user.ID, s.tokenTTL)
	if err != nil {
		return "", fmt.Errorf("service: generate access: %w", err)
	}

	refreshToken := dom.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenString,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour * 15),
	}

	if err := s.repoAuth.SaveRefreshToken(ctx, refreshToken); err != nil {
		return "", err
	}

	return accessTokenString, nil
}

func (s *AuthService) LogoutUser(ctx context.Context, accessToken string) error {
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

	if err := s.repoAuth.DeleteRefreshToken(ctx, claims.UserID); err != nil {
		return fmt.Errorf("db error: %w", err)
	}

	return nil
}

func (s *AuthService) RegisterUser(ctx context.Context, username, email, password string) (dom.User, error) {

	if !utils.ValidatePassword(password) {
		return dom.User{}, customerrors.ErrInvalidInput
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return dom.User{}, err
	}

	res, err := s.repoUser.RegisterUser(ctx, username, email, passwordHash)
	if err != nil {
		return dom.User{}, err
	}

	res = dom.User{
		ID:       res.ID,
		Username: res.Username,
		Email:    res.Email,
	}
	return res, nil

}
