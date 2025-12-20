package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	domUser "main/internal/domain/user"
	"main/internal/pkg/customerrors"
	mwMiddleware "main/internal/server/middleware"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
)

type UserService interface {
	RegisterUser(ctx context.Context, username, email, password string) (domUser.User, error)
	SearchUser(ctx context.Context, query string) ([]domUser.User, error)
}
type AuthService interface {
	LoginUser(ctx context.Context, userID int64, password string) (accessToken string, refreshToken string, err error)
}

type JWTManager interface {
	Exists(context.Context, string) (bool, error)
	Parse(string) (int64, error)
}

type UserHandler struct {
	UserSrv UserService
	AuthSrv AuthService

	upgrader     websocket.Upgrader
	tokenManager JWTManager
	logger       *slog.Logger
}

func NewUserHandler(userSrv UserService,
	authSrv AuthService,
	upgrader websocket.Upgrader,
	tokenManager JWTManager,
	logger *slog.Logger) *UserHandler {
	return &UserHandler{
		UserSrv:      userSrv,
		AuthSrv:      authSrv,
		upgrader:     websocket.Upgrader{},
		tokenManager: tokenManager,
		logger:       logger,
	}
}

func (h *UserHandler) RegisterRoutes(r chi.Router) {

	r.Post("/registration", h.RegisterHandler)
	r.Get("/search", h.usersSearchWS)

	r.Group(func(r chi.Router) {
		r.Use(mwMiddleware.JWTAuth(h.tokenManager))
		r.Get("/search", h.usersSearchWS)
	})
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

		var req string

		if err := json.Unmarshal(message, &req); err != nil {
			h.logger.Error("failed to unmarshal message", slog.String("error", err.Error()))
			return
		}

		result, err := h.UserSrv.SearchUser(r.Context(), req)
		if err != nil {
			h.logger.Error("failed to search users", slog.String("error", err.Error()))
			return
		}
		if len(result) > 0 {
			h.logger.Info("users search results retrieved successfully", slog.String("handler", "usersSearchWS"))
		} else {
			h.logger.Info("no users found", slog.String("handler", "usersSearchWS"))
		}

		err = conn.WriteJSON(result)
		if err != nil {
			h.logger.Error("failed to write message", slog.String("error", err.Error()))
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
		if errors.Is(err, customerrors.ErrInvalidInput) {
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
