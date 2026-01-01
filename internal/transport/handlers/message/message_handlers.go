package message

import (
	"context"
	"encoding/json"
	"log/slog"
	dom "main/internal/domain/entity"
	mwMiddleware "main/internal/server/middleware"
	"main/internal/transport/ws"
	"net/http"

	"github.com/go-chi/chi"
)

type ListDTO struct {
	ChatID      int64 `json:"chat_id"`
	LastMessage int   `json:"last_message"`
}

type MessageService interface {
	SendMessage(ctx context.Context, chatID, senderID int64, senderUsername, text string) (dom.Message, error)
	DeleteMessage(ctx context.Context, messageID int64) error
	EditMessage(ctx context.Context, messageID int64, newText string) error
	ListMessages(ctx context.Context, chatID int64, limit, lastMessage int) ([]dom.Message, error)
}
type ChatService interface {
	CreateChat(ctx context.Context, title string, isPrivate bool, members []int64) (dom.Chat, error)
	AddMembers(ctx context.Context, chatID, userID int64, members []int64) error
	RemoveMember(ctx context.Context, chatID, userID int64) error
	GetChatDetails(ctx context.Context, chatID, userID int64, members []int64) ([]int64, error)
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

	go func() {
		membersId, err := h.ChatSrv.GetChatDetails(r.Context(), request.ChatID, request.SenderID, nil)
		if err != nil {
			h.logger.Error("failed to get chat members", slog.Any("error", err.Error()))
			return
		}

		wsPayload := map[string]interface{}{
			"type": "new_message",
			"data": message,
		}

		for _, memberID := range membersId {
			if memberID == request.SenderID {
				continue
			}
			h.upgrader.WsSendMessage(memberID, wsPayload)
		}

	}()

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

// pattern: /v1/chats/id/messages/{msg_id}
// method:  DELETE
// info:    Удалить сообщение в чате
func (h *MessageHandler) DeleteMessageHandler(w http.ResponseWriter, r *http.Request) {

	var MessageID dom.Message

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&MessageID); err != nil {
		h.logger.Error("failed to decode request", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.MessSrv.DeleteMessage(r.Context(), MessageID.Id); err != nil {
		h.logger.Error("failed to delete message", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *MessageHandler) EditMessage(w http.ResponseWriter, r *http.Request) {
	var request dom.Message

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&request); err != nil {
		h.logger.Error("failed to decode request", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := h.MessSrv.EditMessage(r.Context(), request.Id, request.Text); err != nil {
		h.logger.Error("failed to edit message", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *MessageHandler) ListMessageHandlers(w http.ResponseWriter, r *http.Request) {

	var chatID ListDTO

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&chatID)
	if err != nil {
		h.logger.Error("failed to decode request", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	messages, err := h.MessSrv.ListMessages(r.Context(), chatID.ChatID, 50, chatID.LastMessage)
	if err != nil {
		h.logger.Error("failed to list messages", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := json.MarshalIndent(messages, "", "	")
	if err != nil {
		h.logger.Error("failed to marshal messages", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		h.logger.Error("failed to write response", slog.Any("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
