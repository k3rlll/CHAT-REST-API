package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	domUser "main/internal/domain/user"
	"main/internal/pkg/customerrors"
	mwMiddleware "main/internal/server/middleware"
	srvAuth "main/internal/service/auth"
	srvUser "main/internal/service/user"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
)

type AuthHandler struct {
	UserSrv *srvUser.UserService
	AuthSrv *srvAuth.AuthService
	logger  *slog.Logger
	Manager mwMiddleware.JWTManager
}

func NewAuthHandler(
	userSrv *srvUser.UserService,
	authSrv *srvAuth.AuthService,
	tokenManager mwMiddleware.JWTManager,
	logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		UserSrv: userSrv,
		AuthSrv: authSrv,
		Manager: tokenManager,
		logger:  logger,
	}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	// все пути относительно /auth
	r.Post("/login", h.LoginHandler)

	r.Group(func(r chi.Router) {
		r.Use(mwMiddleware.JWTAuth(h.Manager))
		r.Post("/logout", h.LogoutHandler)
	})
}

/*pattern: /v1/auth/login
method:  POST
info:    Логин по паролю, выдача токенов

succeed:
  - status code: 200 OK
  - response body: JSON { user:{...}, access_token, refresh_token, issued_at }

failed:
  - status code: 400 Bad Request
  - status code: 401 Unauthorized (неверные креды)
  - status code: 429 Too Many Requests
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/

func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var u domUser.User

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	AccessToken, RefreshToken, err := h.AuthSrv.LoginUser(r.Context(), u.ID, u.Password)
	if err != nil {
		if errors.Is(err, customerrors.ErrInvalidNicknameOrPassword) {
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

/*
pattern: /v1/auth/logout
method:  POST
info:    Выход с текущей сессии (инвалидация refresh)

succeed:
  - status code: 204 No Content
  - response body: пусто

failed:
  - status code: 401 Unauthorized
  - status code: 500 Internal Server Error
  - response body: JSON error + time
*/

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

/*
pattern: /v1/auth/logout-all
method:  POST
info:    Выход со всех устройств/сессий

succeed:
  - status code: 204 No Content
  - response body: пусто

failed:
  - status code: 401 Unauthorized
  - status code: 500 Internal Server Error
  - response body: JSON error + time
*/
//TODO: implement logout from all devices
