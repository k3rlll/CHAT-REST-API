package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"main/internal/pkg/customerrors"
	mwMiddleware "main/internal/server/middleware"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
)

type AuthHandler struct {
	AuthSrv AuthService
	logger  *slog.Logger
	Manager JWTManager
}

//go:generate go run github.com/vektra/mockery/v2@v2.32.4 --name=AuthService
type AuthService interface {
	LoginUser(ctx context.Context, userID int64, password string) (accessToken string, refreshToken string, err error)
	LogoutUser(ctx context.Context, userID int64, refreshToken string) error
}

type JWTManager interface {
	Exists(context.Context, string) (bool, error)
	Parse(string) (int64, error)
}

type loginDTO struct {
	ID       int64  `json:"user_id"`
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
		r.Use(mwMiddleware.JWTAuth(h.Manager))
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

	AccessToken, RefreshToken, err := h.AuthSrv.LoginUser(r.Context(), u.ID, u.Password)
	if err != nil {
		if errors.Is(err, customerrors.ErrInvalidInput) {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			h.logger.Info("invalid login attempt", slog.String("user_id", strconv.FormatInt(u.ID, 10)))
			return
		}
		h.logger.Error("failed to login user", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Expires:  time.Now().Add(time.Hour * 24 * 7),
		Value:    RefreshToken,
		HttpOnly: true,
		Path:     "/auth",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Authorization", "Bearer "+AccessToken)

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {

	var user int64

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	c, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tokenString := c.Value

	if err := h.AuthSrv.LogoutUser(r.Context(), user, tokenString); err != nil {
		h.logger.Error("failed to logout user", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
