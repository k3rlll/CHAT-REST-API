package user

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi"

	mwMiddleware "main/internal/delivery/http/middleware/auth"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	"main/internal/pkg/jwt"
)

type UserService interface {
	RegisterUser(ctx context.Context, username, email, password string) (dom.User, error)
	SearchUser(ctx context.Context, query string) ([]dom.User, error)
}

type JWTManager interface {
	Exists(context.Context, string) (bool, error)
	Parse(string) (*jwt.TokenClaims, error)
}

type UserHandler struct {
	UserSrv      UserService
	tokenManager JWTManager
	logger       *slog.Logger
}

func NewUserHandler(userSrv UserService,
	tokenManager JWTManager,
	logger *slog.Logger) *UserHandler {
	return &UserHandler{
		UserSrv:      userSrv,
		tokenManager: tokenManager,
		logger:       logger,
	}
}

type registrationDTO struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserHandler) RegisterRoutes(r chi.Router) {

	r.Post("/registration", h.RegisterHandler)

	r.Group(func(r chi.Router) {
		r.Use(mwMiddleware.JWTAuth(h.tokenManager, h.tokenManager, h.logger))
		r.Get("/search", h.usersSearch)
	})
}

func (h *UserHandler) usersSearch(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "query parameter is required", http.StatusBadRequest)
		return
	}
	users, err := h.UserSrv.SearchUser(r.Context(), query)
	if err != nil {
		h.logger.Error("failed to search users", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	b, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		h.logger.Error("failed to marshal response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", slog.String("error", err.Error()))
		return
	}
}

func (h *UserHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {

	var u registrationDTO

	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	createdUser, err := h.UserSrv.RegisterUser(r.Context(), u.Username, u.Email, u.Password)
	if err != nil {
		if errors.Is(err, customerrors.ErrInvalidInput) {
			http.Error(w, "Password or input invalid", http.StatusUnprocessableEntity)
			h.logger.Info("invalid input during registration")
		} else if errors.Is(err, customerrors.ErrUsernameAlreadyExists) || errors.Is(err, customerrors.ErrEmailAlreadyExists) {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			h.logger.Error("failed to register user", slog.String("error", err.Error()))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	b, err := json.MarshalIndent(createdUser, "", "  ")
	if err != nil {
		h.logger.Error("failed to marshal response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", slog.String("error", err.Error()))
		return
	}
}
