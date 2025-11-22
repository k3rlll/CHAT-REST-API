package handlers

import (
	"log/slog"

	"github.com/go-chi/chi"
)

type HTTPHandler struct {
	UserHandler    *UserHandler
	AuthHandler    *AuthHandler
	ChatHandler    *ChatHandler
	MessageHandler *MessageHandler
	Logger         *slog.Logger
}

func NewHTTPHandler(userHandler *UserHandler, authHandler *AuthHandler, chatHandler *ChatHandler,
	messageHandler *MessageHandler, logger *slog.Logger) *HTTPHandler {
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
