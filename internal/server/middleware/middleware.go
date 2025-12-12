package logger

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/golang-jwt/jwt/v5"
)

type JWTManager interface {
	Parse(accessToken string) (int64, error)
	Exists(ctx context.Context, token string) (bool, error)
}

func New(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		log = log.With(
			slog.String("component", "middleware"),
		)

		log.Info("Middleware initialized")

		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := log.With(
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("request_id", middleware.GetReqID(r.Context())),
				slog.String("path", r.URL.Path),
				slog.String("method", r.Method))

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t1 := time.Now()

			defer func() {
				entry.Info("request completed",
					slog.Int("status", ww.Status()),
					slog.Int("bytes_written", ww.BytesWritten()),
					slog.String("duration", time.Since(t1).String()),
				)
			}()
			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

func JWTAuth(manager JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if strings.EqualFold(r.Header.Get("Connection"), "Upgrade") &&
				strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {

				c, err := r.Cookie("access_token")
				if err != nil {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				tokenString := c.Value
				if exists, err := manager.Exists(r.Context(), tokenString); err != nil || exists {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				userID, err := manager.Parse(tokenString)
				if err != nil {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}

				ctx := context.WithValue(r.Context(), "userID", userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}
			tokenString := parts[1]

			claims := &jwt.RegisteredClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte("mysecretkey"), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), "user", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
