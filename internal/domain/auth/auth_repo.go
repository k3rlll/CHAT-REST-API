package auth

import (
	"context"
)

type TokenRepository interface {
	SaveRefreshToken(ctx context.Context, userID int64, refreshToken string) error
	Login(ctx context.Context, token *TokenPair, userID int64, password string) (*TokenPair, error)
}
