package handlers

import (
	"encoding/json"
	"log/slog"
	domMess "main/internal/domain/message"
	mwMiddleware "main/internal/server/middleware"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
)

type MessageHandler struct {
	UserSrv  UserService
	AuthSrv  AuthService
	MessSrv  MessageService
	ChatSrv  ChatService
	upgrader websocket.Upgrader
	logger   *slog.Logger
	Manager  JWTManager
}

func NewMessageHandler(userSrv UserService,
	authSrv AuthService,
	messSrv MessageService,
	chatSrv ChatService,
	logger *slog.Logger,
	tokenManager JWTManager,
) *MessageHandler {
	return &MessageHandler{
		UserSrv:  userSrv,
		AuthSrv:  authSrv,
		MessSrv:  messSrv,
		ChatSrv:  chatSrv,
		upgrader: websocket.Upgrader{},
		logger:   logger,
		Manager:  tokenManager,
	}
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
	var request domMess.Message

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

	var MessageID domMess.Message

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
	var request domMess.Message

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
	var chatID domMess.Message

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
