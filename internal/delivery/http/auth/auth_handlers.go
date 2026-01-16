package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"

	mwMiddleware "main/internal/delivery/http/middleware/auth"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	"main/internal/pkg/jwt"
)

type AuthHandler struct {
	AuthSrv AuthService
	logger  *slog.Logger
	Manager JWTManager
}

type AuthService interface {
	LoginUser(ctx context.Context, username string, password string) (accessToken string, refreshToken dom.RefreshToken, err error)
	LogoutUser(ctx context.Context, accessToken, refreshToken string) error
}

type JWTManager interface {
	Exists(context.Context, string) (bool, error)
	Parse(string) (*jwt.TokenClaims, error)
}

type loginDTO struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAuthHandler(
	authSrv AuthService,
	tokenManager JWTManager,
	logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		AuthSrv: authSrv,
		Manager: tokenManager,
		logger:  logger,
	}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/login", h.LoginHandler)

	r.Group(func(r chi.Router) {
		r.Use(mwMiddleware.JWTAuth(h.Manager, h.Manager, h.logger))
		r.Post("/logout", h.LogoutHandler)
	})
}

func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var u loginDTO

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	AccessToken, RefreshToken, err := h.AuthSrv.LoginUser(r.Context(), u.Username, u.Password)
	if err != nil {
		if errors.Is(err, customerrors.ErrInvalidInput) {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			h.logger.Info("invalid login attempt", slog.String("username", u.Username))
			return
		}
		h.logger.Error("failed to login user", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Expires:  time.Now().Add(time.Hour * 24 * 15),
		Value:    RefreshToken.Token,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Authorization", "Bearer "+AccessToken)

	json.NewEncoder(w).Encode(map[string]string{
		"access_token": AccessToken,
	})
}

func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "missing authorization header", http.StatusUnauthorized)
		return
	}
	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		http.Error(w, "invalid auth header", http.StatusUnauthorized)
		return
	}
	accessToken := headerParts[1]

	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "refresh token cookie missing", http.StatusBadRequest)
		return
	}
	refreshToken := cookie.Value

	if err := h.AuthSrv.LogoutUser(r.Context(), accessToken, refreshToken); err != nil {
		h.logger.Error("failed to logout user", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	w.WriteHeader(http.StatusOK)
}
