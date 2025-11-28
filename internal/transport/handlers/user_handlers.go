package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	domUser "main/internal/domain/user"
	"main/internal/pkg/customerrors"
	srvAuth "main/internal/service/auth"
	srvChat "main/internal/service/chat"
	srvMessage "main/internal/service/message"
	srvUser "main/internal/service/user"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
)

type UserHandler struct {
	UserSrv  *srvUser.UserService
	AuthSrv  *srvAuth.AuthService
	MessSrv  *srvMessage.MessageService
	ChatSrv  *srvChat.ChatService
	upgrader websocket.Upgrader
	logger   *slog.Logger
}

func NewUserHandler(userSrv *srvUser.UserService,
	authSrv *srvAuth.AuthService,
	messSrv *srvMessage.MessageService,
	chatSrv *srvChat.ChatService,
	upgrader websocket.Upgrader,
	logger *slog.Logger) *UserHandler {
	return &UserHandler{
		UserSrv:  userSrv,
		AuthSrv:  authSrv,
		MessSrv:  messSrv,
		ChatSrv:  chatSrv,
		upgrader: websocket.Upgrader{},
		logger:   logger,
	}
}

func (h *UserHandler) RegisterRoutes(r chi.Router) {

	r.Post("/registration", h.RegisterHandler)
	r.Get("/search", h.usersSearchWS)

	// r.Group(func(r chi.Router) {
	// 	r.Use(mwLogger.JWTAuth)
	// 	r.Get("/search", h.usersSearchWS)
	// })
}

/*pattern: /v1/me
method:  GET
info:    Получить профиль текущего пользователя

succeed:
  - status code: 200 OK
  - response body: JSON { id, username, email?, avatar, ... }

failed:
  - status code: 401 Unauthorized
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/

func meHandler(w http.ResponseWriter, r *http.Request) {}

/*pattern: /v1/users/{id}
method:  GET
info:    Публичный профиль пользователя по ID

succeed:
  - status code: 200 OK
  - response body: JSON { id, username, avatar, status, ... }

failed:
  - status code: 400 Bad Request
  - status code: 403 Forbidden (вы заблокированы друг у друга — по политике)
  - status code: 404 Not Found
  - status code: 500 Internal Server Error
  - response body: JSON error + time*/

func userByIDHandler(w http.ResponseWriter, r *http.Request) {}

/*
pattern: /v1/users
method:  GET
info:    Поиск пользователей; параметры: q, limit, cursor
*/
func (h *UserHandler) usersSearchWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection", slog.String("error", err.Error()))
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			h.logger.Error("failed to read message", slog.String("error", err.Error()))
			http.Error(w, "Failed to read message", http.StatusInternalServerError)
			break
		}
		h.logger.Info("received message", slog.String("message", string(message)))

		result, err := h.UserSrv.SearchUser(r.Context(), string(message))
		if err != nil {
			h.logger.Error("failed to search users", slog.String("error", err.Error()))
			http.Error(w, "Failed to search users", http.StatusInternalServerError)
			break
		}

		b, err := json.MarshalIndent(result, "", "   ")
		if err != nil {
			h.logger.Error("failed to marshal result", slog.String("error", err.Error()))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			break
		}

		response := []byte("Echo: " + string(b))
		err = conn.WriteJSON(response)
		if err != nil {
			h.logger.Error("failed to write message", slog.String("error", err.Error()))
			http.Error(w, "Failed to write message", http.StatusInternalServerError)
			return
		}

	}
}

func (h *UserHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {

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
