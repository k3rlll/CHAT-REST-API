package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"

	mwMiddleware "main/internal/delivery/http/middleware/auth"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	"main/internal/pkg/jwt"
)

type ChatHandler struct {
	MessSrv MessageService
	ChatSrv ChatService
	logger  *slog.Logger
	Manager JWTManager
}

type MessageService interface {
	GetMessages(ctx context.Context, userID, chatID int64, anchorTimeStr string, anchorID string, limit int64) ([]dom.Message, error)
}

type ChatService interface {
	CreateChat(ctx context.Context, title string, isPrivate bool, members []int64) (dom.Chat, error)
	ListOfChats(ctx context.Context, userID int64) ([]dom.Chat, error)
	GetChatDetails(ctx context.Context, chatID int64, userID int64) (dom.Chat, error)
	DeleteChat(ctx context.Context, chatID int64) error
	AddMembers(ctx context.Context, chatID, userID int64, members []int64) error
}

type JWTManager interface {
	Exists(context.Context, string) (bool, error)
	Parse(string) (*jwt.TokenClaims, error)
}

func NewChatHandler(
	messSrv MessageService,
	chatSrv ChatService,
	logger *slog.Logger,
	tokenManager JWTManager,

) *ChatHandler {
	return &ChatHandler{
		MessSrv: messSrv,
		ChatSrv: chatSrv,
		logger:  logger,
		Manager: tokenManager,
	}
}

func (h *ChatHandler) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(mwMiddleware.JWTAuth(h.Manager, h.Manager, h.logger))
		r.Post("/", h.CreateChatHandler)
		r.Get("/", h.GetChatsHandler)
		r.Get("/{chat_id}", h.OpenChatHandler)
		r.Delete("/{chat_id}", h.DeleteChatHandler)
		r.Post("/{chat_id}/members", h.AddMembersHandler)
	})
}

func (h *ChatHandler) CreateChatHandler(w http.ResponseWriter, r *http.Request) {
	var chat dom.Chat

	if err := json.NewDecoder(r.Body).Decode(&chat); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	members := chat.MembersID
	title := chat.Title

	userID, ok := mwMiddleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	hasSelf := false
	for _, m := range members {
		if m == userID {
			hasSelf = true
			break
		}
	}
	if !hasSelf {
		members = append(members, userID)
	}

	createdChat, err := h.ChatSrv.CreateChat(r.Context(), title, chat.IsPrivate, members)
	if err != nil {
		h.logger.Error("failed to create chat", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdChat)
}

func (h *ChatHandler) GetChatsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := mwMiddleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	chats, err := h.ChatSrv.ListOfChats(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get list of chats", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chats); err != nil {
		h.logger.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *ChatHandler) OpenChatHandler(w http.ResponseWriter, r *http.Request) {
	chatIDStr := chi.URLParam(r, "chat_id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}

	userID, ok := mwMiddleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		limit = 50
	}

	anchorID := r.URL.Query().Get("before_id")
	beforeTimeStr := r.URL.Query().Get("before_time")

	chatDetails, err := h.ChatSrv.GetChatDetails(r.Context(), chatID, userID)
	if err != nil {
		h.logger.Error("failed to get chat details", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	messages, err := h.MessSrv.GetMessages(r.Context(), userID, chatID, beforeTimeStr, anchorID, limit)
	if err != nil {
		h.logger.Error("failed to get messages", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Chat     dom.Chat      `json:"chat"`
		Messages []dom.Message `json:"messages"`
	}{
		Chat:     chatDetails,
		Messages: messages,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *ChatHandler) DeleteChatHandler(w http.ResponseWriter, r *http.Request) {
	chatIDStr := chi.URLParam(r, "chat_id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}

	if err := h.ChatSrv.DeleteChat(r.Context(), chatID); err != nil {
		h.logger.Error("failed to delete chat", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ChatHandler) AddMembersHandler(w http.ResponseWriter, r *http.Request) {
	chatIDStr := chi.URLParam(r, "chat_id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}

	userID, ok := mwMiddleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var requestData struct {
		Members []int64 `json:"members"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.ChatSrv.AddMembers(r.Context(), chatID, userID, requestData.Members); err != nil {
		h.logger.Error("failed to add members to chat", slog.String("error", err.Error()))
		switch err {
		case customerrors.ErrUserAlreadyInChat:
			http.Error(w, "conflict", http.StatusConflict)
		case customerrors.ErrInvalidInput:
			http.Error(w, "not found", http.StatusNotFound)
		case customerrors.ErrUserNotMemberOfChat:
			http.Error(w, "no permission", http.StatusForbidden)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
}
