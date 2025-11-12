package auth

import (
	"context"
	"main/internal/pkg/jwt"
)

type TokenRepository interface {
	Login(ctx context.Context, token *jwt.TokenPair, userID int64, password string) (*jwt.TokenPair, error)
	Logout(ctx context.Context, userID int64, token jwt.TokenPair) error
	LogoutAll(ctx context.Context, userID int64) error
}
