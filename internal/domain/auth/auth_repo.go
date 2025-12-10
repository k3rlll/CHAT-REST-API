package auth

import (
	"context"
	dom "main/internal/domain/user"
)

type TokenRepository interface {
	SaveRefreshToken(ctx context.Context, userID int64, refreshToken string) error
	Login(ctx context.Context, token string, userID int64, password string) (dom.User, error)
	GetByEmail(ctx context.Context, email string) (dom.User, error)
}
