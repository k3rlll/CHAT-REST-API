package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	domUser "main/internal/domain/user"
	"main/internal/pkg/customerrors"
	srvAuth "main/internal/service/auth"
	srvUser "main/internal/service/user"
	"net/http"
	"strconv"
	"time"
)

type AuthHandler struct {
	UserSrv       *srvUser.UserService
	AuthSrv       *srvAuth.AuthService
	logger        *slog.Logger
	MaxAtttempts  int
	BlockDuration time.Duration
}

func NewAuthHandler(userSrv *srvUser.UserService, authSrv *srvAuth.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		UserSrv:       userSrv,
		AuthSrv:       authSrv,
		logger:        logger,
		MaxAtttempts:  5,
		BlockDuration: 15 * time.Minute,
	}
}

/*pattern: /v1/auth/registration
method:  POST
info:    Регистрация нового пользователя

succeed:
  - status code: 201 Created
  - response body: JSON { user:{...}, access_token, refresh_token, issued_at }

failed:
  - status code: 400 Bad Request (невалидные поля)
  - status code: 409 Conflict (username/email заняты)
  - status code: 422 Unprocessable Entity (валидация)
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/

// type User struct {
// 	ID       string `json:"id"`
// 	Username string `json:"username"`
// 	Email    string `json:"email"`
// }

func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {

	var u domUser.User

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	createdUser, err := h.UserSrv.RegisterUser(r.Context(), u.Username, u.Email, u.Password)
	if err != nil {
		if errors.Is(err, customerrors.ErrInvalidPassword) {
			http.Error(w, "Password does not meet complexity requirements", http.StatusUnprocessableEntity)
			h.logger.Info("invalid password during registration")
		} else {
			h.logger.Error("failed to register user", slog.String("error", err.Error()))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	b, err := json.MarshalIndent(createdUser, "", "  ")
	if err != nil {
		h.logger.Error("failed to marshal response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

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

	blocked, err := h.AuthSrv.CheckIfBlocked(r.Context(), u.ID)
	if err != nil {
		h.logger.Error("failed to check if user is blocked", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if blocked {
		http.Error(w, "User is blocked due to multiple failed login attempts", http.StatusTooManyRequests)
		h.logger.Info("blocked login attempt", slog.String("user_id", strconv.FormatInt(u.ID, 10)))
		return
	}

	token, err := h.AuthSrv.LoginUser(r.Context(), u.ID, u.Password)
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
		Name:     "access_token",
		Value:    token.AccessToken,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

/*pattern: /v1/auth/token/refresh
method:  POST
info:    Обновление пары токенов по refresh

succeed:
  - status code: 200 OK
  - response body: JSON { access_token, refresh_token, issued_at }

failed:
  - status code: 400 Bad Request
  - status code: 401 Unauthorized (просрочен/заблокирован/невалиден refresh)
  - status code: 429 Too Many Requests
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/

func refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	
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
func logoutHandler(w http.ResponseWriter, r *http.Request) {}

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
func logoutAllHandler(w http.ResponseWriter, r *http.Request) {}
