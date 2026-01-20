package jwt

import (
	"context"
	. "main/pkg/jwt"
)

type contextKey struct{}

var userCtxKey = contextKey{}

func ToContext(ctx context.Context, claims *TokenClaims) context.Context {
	return context.WithValue(ctx, userCtxKey, claims)
}

func FromContext(ctx context.Context) (*TokenClaims, bool) {
	claims, ok := ctx.Value(userCtxKey).(*TokenClaims)
	return claims, ok
}
