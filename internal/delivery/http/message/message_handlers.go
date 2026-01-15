package message

import (
	"context"
	"encoding/json"
	"log/slog"
	mwMiddleware "main/internal/delivery/http/middleware/auth"
	"main/internal/delivery/ws"
	dom "main/internal/domain/entity"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
)

type EditMessageDTO struct {
	MessageID string `json:"message_id"`
	SenderID  int64  `json:"sender_id"`
	ChatID    int64  `json:"chat_id"`
	NewText   string `json:"new_text"`
}

type DeleteMessageDTO struct {
	MessageID []string `json:"message_id"`
	ChatID    int64    `json:"chat_id"`
	UserID    int64    `json:"user_id"`
}

type MessageService interface {
	SendMessage(ctx context.Context, chatID, senderID int64, senderUsername, text string) (*dom.Message, error)
	DeleteMessage(ctx context.Context, senderID int64, chatID int64, msgID []string) error
	EditMessage(ctx context.Context, senderID int64, chatID int64, msgID string, newText string) error
	GetMessages(ctx context.Context, userID, chatID int64, anchorTimeStr string, anchorID string, limit int64) ([]dom.Message, error)
}
type ChatService interface {
	CreateChat(ctx context.Context, title string, isPrivate bool, members []int64) (dom.Chat, error)
	AddMembers(ctx context.Context, chatID, userID int64, members []int64) error
	RemoveMember(ctx context.Context, chatID, userID int64) error
	GetChatDetails(ctx context.Context, chatID int64, userID int64) (dom.Chat, error)
}

type JWTManager interface {
	Exists(context.Context, string) (bool, error)
	Parse(string) (int64, error)
}

type MessageHandler struct {
	MessSrv  MessageService
	ChatSrv  ChatService
	logger   *slog.Logger
	upgrader *ws.Manager
	Manager  JWTManager
}

func NewMessageHandler(
	messSrv MessageService,
	chatSrv ChatService,
	logger *slog.Logger,
	upgrader *ws.Manager,
	tokenManager JWTManager,
) *MessageHandler {
	return &MessageHandler{
		MessSrv:  messSrv,
		ChatSrv:  chatSrv,
		logger:   logger,
		upgrader: upgrader,
		Manager:  tokenManager,
	}
}

// /chats/{id}/messages
func (h *MessageHandler) RegisterRoutes(r chi.Router) {

	r.Group(func(r chi.Router) {
		r.Use(mwMiddleware.JWTAuth(h.Manager))
		r.Post("/", h.SendMessage)
		r.Delete("/{msg_id}", h.DeleteMessageHandler)
		r.Get("/", h.ListMessageHandlers)
		r.Put("/{msg_id}", h.EditMessage)
	})

	r.Get("/ws", h.ConnectWebSocket)
}

func (h *MessageHandler) ConnectWebSocket(w http.ResponseWriter, r *http.Request) {
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	userID, err := h.Manager.Parse(tokenString)
	if err != nil {
		h.logger.Error("ws auth failed", "error", err)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	h.upgrader.HandleConnection(w, r, userID)
}

// pattern: /v1/chats/id/messages
// method:  POST
// info:    Отправить сообщение в чат
func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var request dom.Message

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&request); err != nil {
		h.logger.Error("failed to decode request", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	message, err := h.MessSrv.SendMessage(r.Context(),
		request.ChatID,
		request.SenderID,
		request.SenderUsername,
		request.Text)
	if err != nil {
		h.logger.Error("failed to send message", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	go func(ctx context.Context, chatID int64, senderID int64) {
		membersId, err := h.ChatSrv.GetChatDetails(ctx, chatID, senderID)
		if err != nil {
			h.logger.Error("failed to get chat members", slog.Any("error", err.Error()))
			return
		}

		wsPayload := map[string]interface{}{
			"type": "new_message",
			"data": message,
		}

		for _, memberID := range membersId.MembersID {
			if memberID == request.SenderID {
				continue
			}
			h.upgrader.WsUnicast(memberID, wsPayload)
		}

	}(context.Background(), request.ChatID, request.SenderID)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

// pattern: /v1/chats/id/messages/{msg_id}
// method:  DELETE
// info:    Delete message by ID
func (h *MessageHandler) DeleteMessageHandler(w http.ResponseWriter, r *http.Request) {

	var request DeleteMessageDTO

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&request); err != nil {
		h.logger.Error("failed to decode request", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.MessSrv.DeleteMessage(r.Context(), request.UserID, request.ChatID, request.MessageID); err != nil {
		h.logger.Error("failed to delete message", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	go func(chatID int64, userID int64) {
		bctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		membersId, err := h.ChatSrv.GetChatDetails(bctx, chatID, userID)
		if err != nil {
			h.logger.Error("failed to get chat members", slog.Any("error", err.Error()))
			return
		}
		wsPayload := map[string]interface{}{
			"type": "delete_message",
			"data": request,
		}
		for _, memberID := range membersId.MembersID {
			h.upgrader.WsUnicast(memberID, wsPayload)
		}

	}(request.ChatID, request.UserID)

	w.WriteHeader(http.StatusOK)
}

// pattern: /v1/chats/id/messages/{msg_id}
// method:  PUT
// info:    Edit message by ID
func (h *MessageHandler) EditMessage(w http.ResponseWriter, r *http.Request) {
	var request EditMessageDTO

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&request); err != nil {
		h.logger.Error("failed to decode request", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.MessSrv.EditMessage(r.Context(), request.SenderID, request.ChatID, request.MessageID, request.NewText); err != nil {
		h.logger.Error("failed to edit message", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

	go func(ctx context.Context, chatID int64, userID int64) {
		membersId, err := h.ChatSrv.GetChatDetails(ctx, chatID, userID)
		if err != nil {
			h.logger.Error("failed to get chat members", slog.Any("error", err.Error()))
			return
		}

		wsPayload := map[string]interface{}{
			"type": "edit_message",
			"data": request,
		}
		for _, memberID := range membersId.MembersID {
			if memberID == userID {
				continue
			}
			h.upgrader.WsUnicast(memberID, wsPayload)
		}
	}(context.Background(), request.ChatID, request.SenderID)

}

func (h *MessageHandler) ListMessageHandlers(w http.ResponseWriter, r *http.Request) {
	userID, err := mwMiddleware.GetUserIDFromContext(r.Context())
	if err != nil {
		h.logger.Error("failed to get user id from context", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	chatIDStr := chi.URLParam(r, "id")
	lastMsgStr := r.URL.Query().Get("last_message")
	lastMsgID := r.URL.Query().Get("last_message_id")

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		h.logger.Error("failed to parse chat id", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	messages, err := h.MessSrv.GetMessages(r.Context(), userID, chatID, lastMsgStr, lastMsgID, 50)
	if err != nil {
		h.logger.Error("failed to list messages", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(messages)
	if err != nil {
		h.logger.Error("failed to encode response", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
