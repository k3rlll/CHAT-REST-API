package auth

import (
	"context"
	"fmt"
	rdb "main/internal/database/redis"
	db "main/internal/domain/user"
	customerrors "main/internal/pkg/customerrors"
	jwt "main/internal/pkg/jwt"
	"time"

	"golang.org/x/crypto/bcrypt"
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

type AuthRepository interface {
	GetPasswordHash(ctx context.Context, refreshToken string, userID int64, password string) (db.User, error)
	SaveRefreshToken(ctx context.Context, userID int64, refreshToken string) error
	DeleteRefreshToken(ctx context.Context, userID int64) error
}

type SetInterface interface {
	Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error
}

type Token interface {
	NewAccessToken(userID int64, ttl time.Duration) (string, error)
	NewRefreshToken() (string, error)
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

func (s *AuthService) LoginUser(ctx context.Context,
	userID int64,
	password string) (
	AccessToken string,
	RefreshToken string,
	err error) {

	user, err := s.Repo.GetPasswordHash(ctx, RefreshToken, userID, password)
	if err != nil {
		return "", "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
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
