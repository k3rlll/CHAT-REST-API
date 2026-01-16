package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"main/internal/pkg/jwt"
)

type ContextKey string

const UserIDKey ContextKey = "user_id"

type TokenParser interface {
	Parse(accessToken string) (*jwt.TokenClaims, error)
}

type TokenBlacklistChecker interface {
	Exists(ctx context.Context, key string) (bool, error)
}

func JWTAuth(parser TokenParser, blacklist TokenBlacklistChecker, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenString string

			if isWebSocket(r) {
				c, err := r.Cookie("access_token")
				if err != nil {
					http.Error(w, "missing cookie", http.StatusUnauthorized)
					return
				}
				tokenString = c.Value
			} else {
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					http.Error(w, "missing authorization header", http.StatusUnauthorized)
					return
				}
				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || parts[0] != "Bearer" {
					http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
					return
				}
				tokenString = parts[1]
			}

			isBanned, err := blacklist.Exists(r.Context(), tokenString)
			if err != nil {
				log.Error("failed to check blacklist", slog.String("error", err.Error()))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if isBanned {
				http.Error(w, "token is revoked", http.StatusUnauthorized)
				return
			}

			claims, err := parser.Parse(tokenString)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}

func isWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Connection"), "Upgrade") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}
