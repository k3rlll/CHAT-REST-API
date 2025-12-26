package message

import (
	"context"
	"encoding/json"
	"log/slog"
	dom "main/internal/domain/entity"
	mwMiddleware "main/internal/server/middleware"
	"net/http"

	"github.com/go-chi/chi"
)

type MessageService interface {
	SendMessage(ctx context.Context, chatID, senderID int64, senderUsername, text string) (dom.Message, error)
	DeleteMessage(ctx context.Context, messageID int64) error
	EditMessage(ctx context.Context, messageID int64, newText string) error
	ListMessages(ctx context.Context, chatID int64) ([]dom.Message, error)
}
type ChatService interface {
	CreateChat(ctx context.Context, title string, isPrivate bool, members []int64) (dom.Chat, error)
	AddMembers(ctx context.Context, chatID, userID int64, members []int64) error
	RemoveMember(ctx context.Context, chatID, userID int64) error
}

type JWTManager interface {
	Exists(context.Context, string) (bool, error)
	Parse(string) (int64, error)
}

type MessageHandler struct {
	MessSrv MessageService
	ChatSrv ChatService
	logger  *slog.Logger
	Manager JWTManager
}

func NewMessageHandler(
	messSrv MessageService,
	chatSrv ChatService,
	logger *slog.Logger,
	tokenManager JWTManager,
) *MessageHandler {
	return &MessageHandler{
		MessSrv: messSrv,
		ChatSrv: chatSrv,
		logger:  logger,
		Manager: tokenManager,
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
		r.Post("/add_member", h.AddMembers)
		r.Post("/create_chat", h.CreateChat)
	})
}

func (h *MessageHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Title     string  `json:"title"`
		IsPrivate bool    `json:"is_private"`
		Members   []int64 `json:"members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Error("failed to decode request", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	chat, err := h.ChatSrv.CreateChat(r.Context(), request.Title, request.IsPrivate, request.Members)
	if err != nil {
		h.logger.Error("failed to create chat", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	b, err := json.MarshalIndent(chat, "", "	")
	if err != nil {
		h.logger.Error("failed to marshal chat", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		h.logger.Error("failed to write response", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// pattern: /v1/chats/id/messages
// method:  POST
// info:    Отправить сообщение в чат
func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var request dom.Message

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Error("failed to decode request", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	message, err := h.MessSrv.SendMessage(r.Context(),
		request.ChatID,
		request.SenderID,
		request.SenderUsername,
		request.Text)
	if err != nil {
		h.logger.Error("failed to send message", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	b, err := json.MarshalIndent(message, "", "	")
	if err != nil {
		h.logger.Error("failed to marshal message", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		h.logger.Error("failed to write response", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// pattern: /v1/chats/id/messages/{msg_id}
// method:  DELETE
// info:    Удалить сообщение в чате
func (h *MessageHandler) DeleteMessageHandler(w http.ResponseWriter, r *http.Request) {

	var MessageID dom.Message

	if err := json.NewDecoder(r.Body).Decode(&MessageID); err != nil {
		h.logger.Error("failed to decode request", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.MessSrv.DeleteMessage(r.Context(), MessageID.Id); err != nil {
		h.logger.Error("failed to delete message", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// pattern: /v1/chats/id/messages/{msg_id}
// method:  PUT
// info:    Отредактировать сообщение в чате
func (h *MessageHandler) EditMessage(w http.ResponseWriter, r *http.Request) {
	var request dom.Message

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Error("failed to decode request", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := h.MessSrv.EditMessage(r.Context(), request.Id, request.Text); err != nil {
		h.logger.Error("failed to edit message", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// pattern: /v1/chats/id/messages
// method:  GET
// info:    Лист сообщений в чате
func (h *MessageHandler) ListMessageHandlers(w http.ResponseWriter, r *http.Request) {
	var chatID dom.Message

	if err := json.NewDecoder(r.Body).Decode(&chatID); err != nil {
		h.logger.Error("failed to decode request", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	messages, err := h.MessSrv.ListMessages(r.Context(), chatID.Id)
	if err != nil {
		h.logger.Error("failed to list messages", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := json.MarshalIndent(messages, "", "	")
	if err != nil {
		h.logger.Error("failed to marshal messages", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		h.logger.Error("failed to write response", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *MessageHandler) AddMembers(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ChatID  int64   `json:"chat_id"`
		UserID  int64   `json:"user_id"`
		Members []int64 `json:"members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Error("failed to decode request", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.ChatSrv.AddMembers(r.Context(), request.ChatID, request.UserID, request.Members); err != nil {
		h.logger.Error("failed to add members", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
