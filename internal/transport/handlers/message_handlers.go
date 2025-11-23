package handlers

import (
	"encoding/json"
	"log/slog"
	domMess "main/internal/domain/message"
	srvAuth "main/internal/service/auth"
	srvChat "main/internal/service/chat"
	srvMessage "main/internal/service/message"
	srvUser "main/internal/service/user"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
)

type MessageHandler struct {
	UserSrv  *srvUser.UserService
	AuthSrv  *srvAuth.AuthService
	MessSrv  *srvMessage.MessageService
	ChatSrv  *srvChat.ChatService
	upgrader websocket.Upgrader
	logger   *slog.Logger
}

func NewMessageHandler(userSrv *srvUser.UserService,
	authSrv *srvAuth.AuthService,
	messSrv *srvMessage.MessageService,
	chatSrv *srvChat.ChatService,
	logger *slog.Logger) *MessageHandler {
	return &MessageHandler{
		UserSrv:  userSrv,
		AuthSrv:  authSrv,
		MessSrv:  messSrv,
		ChatSrv:  chatSrv,
		upgrader: websocket.Upgrader{},
		logger:   logger,
	}
}

// /chats/{id}/messages
func (h *MessageHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.Send)
	r.Delete("/{msg_id}", h.DeleteMessageHandler)
	r.Get("/", h.ListMessageHandlers)
	r.Put("/{msg_id}", h.EditMessage)
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

	message, err := h.MessSrv.Send(r.Context(), request.ChatID, request.SenderID, request.Text)
	if err != nil {
		h.logger.Error("failed to send message", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
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
