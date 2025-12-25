package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	dom "main/internal/domain/entity"
	mwMiddleware "main/internal/server/middleware"
	"net/http"

	"github.com/go-chi/chi"
)

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

type MessageService interface {
	Send(ctx context.Context, chatID, senderID int64, senderUsername, text string) (dom.Message, error)
	DeleteMessage(ctx context.Context, messageID int64) error
	Edit(ctx context.Context, messageID int64, newText string) error
	List(ctx context.Context, chatID int64) ([]dom.Message, error)
}
type ChatService interface {
	CreateChat(ctx context.Context, name string, memberIDs []int64) (dom.Chat, error)
	AddMember(ctx context.Context, chatID, userID int64) error
	RemoveMember(ctx context.Context, chatID, userID int64) error
}

type JWTManager interface {
	Exists(context.Context, string) (bool, error)
	Parse(string) (int64, error)
}

// /chats/{id}/messages
func (h *MessageHandler) RegisterRoutes(r chi.Router) {

	r.Group(func(r chi.Router) {
		r.Use(mwMiddleware.JWTAuth(h.Manager))
		r.Post("/", h.Send)
		r.Delete("/{msg_id}", h.DeleteMessageHandler)
		r.Get("/", h.ListMessageHandlers)
		r.Put("/{msg_id}", h.EditMessage)
	})
}

// pattern: /v1/chats/id/messages
// method:  POST
// info:    Отправить сообщение в чат
func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	var request dom.Message

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Error("failed to decode request", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	message, err := h.MessSrv.Send(r.Context(),
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
	if err := h.MessSrv.Edit(r.Context(), request.Id, request.Text); err != nil {
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

	messages, err := h.MessSrv.List(r.Context(), chatID.Id)
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
