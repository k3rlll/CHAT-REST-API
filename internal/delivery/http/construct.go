package handlers

import (
	"log/slog"

	auth "main/internal/delivery/http/auth"
	chat "main/internal/delivery/http/chat"
	message "main/internal/delivery/http/message"
	user "main/internal/delivery/http/user"

	"github.com/go-chi/chi"
)

type HTTPHandler struct {
	UserHandler    *user.UserHandler
	AuthHandler    *auth.AuthHandler
	ChatHandler    *chat.ChatHandler
	MessageHandler *message.MessageHandler
	Logger         *slog.Logger
}

func NewHTTPHandler(userHandler *user.UserHandler, authHandler *auth.AuthHandler, chatHandler *chat.ChatHandler,
	messageHandler *message.MessageHandler, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		UserHandler:    userHandler,
		AuthHandler:    authHandler,
		ChatHandler:    chatHandler,
		MessageHandler: messageHandler,
		Logger:         logger,
	}
}

func (h *HTTPHandler) RegisterRoutes(r chi.Router) {

	r.Route("/users", func(r chi.Router) {
		h.UserHandler.RegisterRoutes(r)
	})

	r.Route("/auth", func(r chi.Router) {
		h.AuthHandler.RegisterRoutes(r)
	})

	r.Route("/chats", func(r chi.Router) {
		h.ChatHandler.RegisterRoutes(r)
	})

	r.Route("/messages", func(r chi.Router) {
		h.MessageHandler.RegisterRoutes(r)
	})
}
