package cmd

import (
	"context"
	"log/slog"
	"main/internal/config"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	db "main/internal/database"
	auth "main/internal/database/auth_repo"
	chat "main/internal/database/chat_repo"
	msg "main/internal/database/message_repo"
	user "main/internal/database/user_repo"
	mwLogger "main/internal/server/logger"
	srvAuth "main/internal/service/auth"
	srvUser "main/internal/service/user"
	HTTP "main/internal/transport/handlers"

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
	addr := net.JoinHostPort(cfg.Server.Host, string(cfg.Server.Port))

	router := chi.NewRouter()

	dbConn, err := db.NewDBPool(cfg.DatabaseDSN())
	authRepo := auth.NewTokenRepository(dbConn, logger)
	userRepo := user.NewUserRepository(dbConn, logger)
	chatRepo := chat.NewChatRepository(dbConn, logger)
	msgRepo := msg.NewMessageRepository(dbConn, logger)

	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		return
	}
	userService := srvUser.NewUserService(&userRepo, logger)
	authService := srvAuth.NewAuthService(&authRepo, logger)
	authHandler := HTTP.NewAuthHandler(userService, authService, logger)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	serverParams := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.Timeout,
		WriteTimeout: cfg.Server.Timeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	router.Use(middleware.RequestID)
	router.Use(mwLogger.New(logger))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(mwLogger.JWTAuth)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http.frontend.com", "http://localhost:8082"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

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

	//TODO:close database connection

	logger.Info("server stopped")
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
