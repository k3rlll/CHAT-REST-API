package user

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi"

	mwMiddleware "main/internal/delivery/http/middleware/auth"
	dom "main/internal/domain/entity"
	"main/pkg/jwt"
)

type UserService interface {
	SearchUser(ctx context.Context, query string) ([]dom.User, error)
}

type JWTParser interface {
	Exists(context.Context, string) (bool, error)
	Parse(string) (*jwt.TokenClaims, error)
}

type UserHandler struct {
	UserSrv     UserService
	tokenParser JWTParser
	logger      *slog.Logger
}

func NewUserHandler(userSrv UserService,
	tokenParser JWTParser,
	logger *slog.Logger) *UserHandler {
	return &UserHandler{
		UserSrv:     userSrv,
		tokenParser: tokenParser,
		logger:      logger,
	}
}

type registrationDTO struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserHandler) RegisterRoutes(r chi.Router) {

	r.Group(func(r chi.Router) {
		r.Use(mwMiddleware.JWTAuth(h.tokenParser, h.tokenParser, h.logger))
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
