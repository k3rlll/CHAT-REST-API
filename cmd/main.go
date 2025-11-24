package main

import (
	"context"
	"log/slog"
	"main/internal/config"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	db "main/internal/database"
	auth "main/internal/database/auth_repo"
	chat "main/internal/database/chat_repo"
	msg "main/internal/database/message_repo"
	user "main/internal/database/user_repo"
	mwLogger "main/internal/server/logger"
	srvAuth "main/internal/service/auth"
	srvChat "main/internal/service/chat"
	srvMessage "main/internal/service/message"
	srvUser "main/internal/service/user"
	httpHandler "main/internal/transport/handlers"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

const (
	envLocal = "local"
	envProd  = "prod"
	envDev   = "dev"
)

func main() {
	cfg := config.MustLoadConfig()

	logger := setupLogger(cfg.Env)

	logger.Info("Starting application", slog.String("env", cfg.Env))
	addr := net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port))

	router := chi.NewRouter()

	dbConn, err := db.NewDBPool(cfg.DatabaseDSN())
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		return
	}
	defer dbConn.Close()

	if err := dbConn.Ping(context.Background()); err != nil {
		logger.Error("failed to ping database", slog.String("error", err.Error()))
		return
	}

	logger.Info("Connected to database successfully")

	router.Use(middleware.RequestID)
	router.Use(mwLogger.New(logger))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http.frontend.com", "http://localhost:8082"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	authRepo := auth.NewTokenRepository(dbConn, logger)
	userRepo := user.NewUserRepository(dbConn, logger)
	chatRepo := chat.NewChatRepository(dbConn, logger)
	msgRepo := msg.NewMessageRepository(dbConn, logger)

	userService := srvUser.NewUserService(userRepo, logger)
	authService := srvAuth.NewAuthService(authRepo, logger)
	chatService := srvChat.NewChatService(userRepo, chatRepo, logger)
	messageService := srvMessage.NewMessageService(chatRepo, msgRepo, logger)

	userHandler := httpHandler.NewUserHandler(userService, authService, messageService, chatService, logger)
	authHandler := httpHandler.NewAuthHandler(userService, authService, logger)
	chatHandler := httpHandler.NewChatHandler(userService, authService, messageService, chatService, logger)
	messageHandler := httpHandler.NewMessageHandler(userService, authService, messageService, chatService, logger)

	HTTP := httpHandler.NewHTTPHandler(userHandler, authHandler, chatHandler, messageHandler, logger)

	HTTP.RegisterRoutes(router)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	serverParams := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.Timeout,
		WriteTimeout: cfg.Server.Timeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		if err := serverParams.ListenAndServe(); err != nil {
			logger.Error("HTTP server stopped", slog.String("error", err.Error()))
		}
	}()

	logger.Info("HTTP server is started")

	<-done
	logger.Info("stopping server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := serverParams.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed", slog.String("error", err.Error()))
	}

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envLocal, envDev:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	default:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}
