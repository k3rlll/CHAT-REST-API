package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	dom "main/internal/domain/entity"

	"main/internal/pkg/customerrors"
	mwMiddleware "main/internal/server/middleware"
	"net/http"

	"github.com/go-chi/chi"
)

type ChatHandler struct {
	MessSrv MessageService
	ChatSrv ChatService
	logger  *slog.Logger
	Manager JWTManager
}

type UserService interface {
	RegisterUser(ctx context.Context, username, email, password string) (dom.User, error)
	SearchUser(ctx context.Context, query string) ([]dom.User, error)
}

type MessageService interface {
	SendMessage(ctx context.Context, chatID int64, userID int64, senderUsername string, text string) (dom.Message, error)
	DeleteMessage(ctx context.Context, msgID int64) error
	ListMessages(ctx context.Context, chatID int64) ([]dom.Message, error)
}

type ChatService interface {
	CreateChat(ctx context.Context, isGroup bool, title string, members []int64) (dom.Chat, error)
	ListOfChats(ctx context.Context, userID int64) ([]dom.Chat, error)
	OpenChat(ctx context.Context, chatID int64, userID int64) (dom.Chat, []dom.Message, error)
	GetChatDetails(ctx context.Context, chatID int64, userID int64) (dom.Chat, error)
	DeleteChat(ctx context.Context, chatID int64) error
	AddMembers(ctx context.Context, chatID, userID int64, members []int64) error
}

type JWTManager interface {
	Exists(context.Context, string) (bool, error)
	Parse(string) (int64, error)
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

// все пути относительно /chats
func (h *ChatHandler) RegisterRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(mwMiddleware.JWTAuth(h.Manager))
		r.Post("/", h.CreateChatHandler)
		r.Get("/", h.GetChatsHandler)
		r.Get("/{id}", h.OpenChatHandler)
		r.Delete("/{id}", h.DeleteChatHandler)
		r.Post("/{id}/members", h.AddMembersHandler)
	})
}

/*pattern: /v1/chats
method:  POST
info:    Создать чат: direct (user_id) или group (title)
*/

func (h *ChatHandler) CreateChatHandler(w http.ResponseWriter, r *http.Request) {

	var chat dom.Chat

	if err := json.NewDecoder(r.Body).Decode(&chat); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	members := chat.MembersID
	title := chat.Title
	_, err := h.ChatSrv.CreateChat(r.Context(), false, title, members)
	if err != nil {
		h.logger.Error("failed to create chat", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// pattern: /v1/chats
// method:  GET
// info:    Список чатов пользователя; параметры: cursor, limit, q

func (h *ChatHandler) GetChatsHandler(w http.ResponseWriter, r *http.Request) {

	var userId int64
	if err := json.NewDecoder(r.Body).Decode(&userId); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chats, err := h.ChatSrv.ListOfChats(r.Context(), userId)
	if err != nil {
		h.logger.Error("failed to get list of chats", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b, err := json.MarshalIndent(chats, "", "  ")
	if err != nil {
		h.logger.Error("failed to marshal response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// pattern: /v1/chats/{id}
// method:  GET
// info:    Детали чата (если участник)

func (h *ChatHandler) OpenChatHandler(w http.ResponseWriter, r *http.Request) {

	var requestData map[string]int64
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	chatID := requestData["chat_id"]
	userID := requestData["user_id"]

	chat, messages, err := h.ChatSrv.OpenChat(r.Context(), chatID, userID)
	if err != nil {
		h.logger.Error("failed to open chat", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	marshalDetails, err := json.MarshalIndent(chat, "", "  ")
	if err != nil {
		h.logger.Error("failed to marshal response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	marshalMessages, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		h.logger.Error("failed to marshal response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := struct {
		Chat     json.RawMessage `json:"chat"`
		Messages json.RawMessage `json:"messages"`
	}{
		Chat:     marshalDetails,
		Messages: marshalMessages,
	}

	b, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		h.logger.Error("failed to marshal response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(b); err != nil {
		h.logger.Error("failed to write response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ChatHandler) DeleteChatHandler(w http.ResponseWriter, r *http.Request) {

	var chat_id int64

	if err := json.NewDecoder(r.Body).Decode(&chat_id); err != nil {
		h.logger.Error("failed to decode request", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.ChatSrv.DeleteChat(r.Context(), chat_id); err != nil {
		h.logger.Error("failed to delete chat", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

// pattern: /v1/chats/{id}/members
// method:  GET
// info:    Список участников чата

// succeed:
//   - status code: 200 OK
//   - response body: JSON { items:[{ user, role }], total }

// failed:
//   - status code: 401 Unauthorized
//   - status code: 403 Forbidden
//   - status code: 404 Not Found
//   - status code: 500 Internal Server Error
//   - response body: JSON error + time

// func getListMembersHandler(w http.ResponseWriter, r *http.Request) {}

// pattern: /v1/chats/{id}/members
// method:  POST
// info:    Добавить участника(ов) в групповой чат

func (h *ChatHandler) AddMembersHandler(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		ChatID  int64   `json:"chat_id"`
		Members []int64 `json:"members"`
		UserID  int64   `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	chatID := requestData.ChatID
	members := requestData.Members
	userID := requestData.UserID

	if err := h.ChatSrv.AddMembers(r.Context(), chatID, userID, members); err != nil {
		h.logger.Error("failed to add members to chat", slog.String("error", err.Error()))
		if err == customerrors.ErrUserAlreadyInChat {
			http.Error(w, "conflict", http.StatusConflict)
			return
		} else if err == customerrors.ErrInvalidInput {
			http.Error(w, "not found", http.StatusNotFound)
			return
		} else if err == customerrors.ErrUserNotMemberOfChat {
			http.Error(w, "no permission", http.StatusForbidden)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
}
