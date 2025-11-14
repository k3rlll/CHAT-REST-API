package auth

import (
	"context"
)

type TokenRepository interface {
	Login(ctx context.Context, token *TokenPair, userID int64, password string) (*TokenPair, error)
	Logout(ctx context.Context, userID int64, token *TokenPair) error
	LogoutAll(ctx context.Context, userID int64) error
}
