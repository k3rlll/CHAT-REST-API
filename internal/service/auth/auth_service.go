package auth

import (
	"context"
	"errors"
	"fmt"
	rdb "main/internal/database/redis"
	dom "main/internal/domain/entity"
	customerrors "main/internal/pkg/customerrors"
	jwt "main/internal/pkg/jwt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAlreadyLoggedIn = errors.New("user already logged in")
)

type JWTFacade struct {
	Parser    *jwt.Claims
	redisRepo *rdb.Cache
}

type AuthService struct {
	redis SetInterface
	jwt   Token
	Repo  AuthRepository
}

//go:generate mockgen -source=auth_service.go -destination=./mock/auth_mocks.go -package=mock
type AuthRepository interface {
	GetPasswordHash(ctx context.Context, userID int64, password string) (dom.User, error)
	SaveRefreshToken(ctx context.Context, refreshToken dom.RefreshToken) error
	GetRefreshToken(ctx context.Context, userID int64) (string, error)
	DeleteRefreshToken(ctx context.Context, userID int64) error
}

type SetInterface interface {
	Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error
}

type Token interface {
	NewAccessToken(userID int64, ttl time.Duration) (string, error)
	NewRefreshToken() (string, error)
	Parse(accessToken string) (int64, error)
}

func NewTokenService() Token {
	return &jwt.Claims{}
}

func NewAuthService(repo AuthRepository, jwt Token, redis SetInterface) *AuthService {
	return &AuthService{
		redis: redis,
		jwt:   jwt,
		Repo:  repo,
	}
}

func NewJWTFacade(parser *jwt.Claims, redisRepo *rdb.Cache) *JWTFacade {
	return &JWTFacade{
		Parser:    parser,
		redisRepo: redisRepo,
	}
}

func (s *AuthService) LoginUser(ctx context.Context, userID int64, password string) (string, dom.RefreshToken, error) {

	user, err := s.Repo.GetPasswordHash(ctx, userID, password)
	if err != nil {
		return "", dom.RefreshToken{}, customerrors.ErrDatabase
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", dom.RefreshToken{}, customerrors.ErrInvalidInput
	}
	storedRefreshToken, err := s.Repo.GetRefreshToken(ctx, userID)
	if err != nil {
		return "", dom.RefreshToken{}, fmt.Errorf("failed to get refresh token: %w", err)
	}

	accessToken, err := s.jwt.NewAccessToken(userID, time.Minute*15)
	if err != nil {
		return "", dom.RefreshToken{}, err
	}
	if storedRefreshToken != "" {
		storedRefreshToken, err = s.jwt.NewRefreshToken()
		if err != nil {
			return "", dom.RefreshToken{}, err
		}
	}
	res := dom.RefreshToken{
		UserID:    userID,
		Token:     storedRefreshToken,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add((24 * time.Hour) * 15),
	}

	err = s.Repo.SaveRefreshToken(ctx, res)
	if err != nil {
		return "", dom.RefreshToken{}, err
	}

	return accessToken, res, nil
}

func (s *AuthService) LogoutUser(ctx context.Context, userID int64, accessToken string) error {

	if userID == 0 || userID < 0 {
		return fmt.Errorf("invalid userID")
	}

	if accessToken == "" {
		return fmt.Errorf("access token is empty")
	}

	err := s.redis.Set(ctx, accessToken, "blacklist", 60*15)
	if err != nil {
		return fmt.Errorf("failed to blacklist access token: %w", err)
	}

	err = s.Repo.DeleteRefreshToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	return nil
}

func (j *JWTFacade) Parse(accessToken string) (int64, error) {
	return j.Parser.Parse(accessToken)
}

func (j *JWTFacade) Exists(ctx context.Context, token string) (bool, error) {
	return j.redisRepo.Exists(ctx, token)
}
