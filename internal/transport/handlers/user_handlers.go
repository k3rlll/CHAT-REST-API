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
	"strconv"

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

func (h *UserHandler) usersSearchWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection", slog.String("error", err.Error()))
		h.logger.Info("handler", slog.String("handler", "usersSearchWS"))
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			h.logger.Error("failed to read message", slog.String("error", err.Error()))
			return
		}
		h.logger.Info("received message", slog.String("message", string(message)))

		result, err := h.UserSrv.SearchUser(r.Context(), string(message))
		if err != nil {
			h.logger.Error("failed to search users", slog.String("error", err.Error()))
			return
		}
		if len(result) > 0 {
			h.logger.Info("users search results retrieved successfully",
				slog.Int("count", len(result)),
				slog.String("first_user_username", result[0].Nickname),
				slog.String("first_user_id", strconv.Itoa(int(result[0].ID))),
				slog.String("handler", "usersSearchWS"))
		} else {
			h.logger.Info("no users found", slog.String("handler", "usersSearchWS"))
		}

		b, err := json.MarshalIndent(result, "", "   ")
		if err != nil {
			h.logger.Error("failed to marshal result", slog.String("error", err.Error()))
			return
		}
		h.logger.Info("marshaled search result", slog.String("result", string(b)))

		response := map[string]interface{}{
			"echo": string(b),
		}
		h.logger.Info("sending response", slog.String("response", string(b)))

		err = conn.WriteJSON(response) // Используем WriteJSON для отправки данных в формате JSON
		if err != nil {
			h.logger.Error("failed to write message", slog.String("error", err.Error()))
			return // Завершаем соединение при ошибке
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
	createdUser, err := h.UserSrv.RegisterUser(r.Context(), u.Nickname, u.Email, u.Password)
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
