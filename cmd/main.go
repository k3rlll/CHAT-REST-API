package main

import (
	"context"
	"log/slog"
	config "main/internal/config"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	psql "main/internal/database/postgres"
	auth "main/internal/database/postgres/auth_repo"
	chat "main/internal/database/postgres/chat_repo"
	msg "main/internal/database/postgres/message_repo"
	user "main/internal/database/postgres/user_repo"
	rdb "main/internal/database/redis"
	claims "main/internal/pkg/jwt"
	mwMiddleware "main/internal/server/middleware"
	srvAuth "main/internal/service/auth"
	srvChat "main/internal/service/chat"
	srvMessage "main/internal/service/message"
	srvUser "main/internal/service/user"
	httpHandler "main/internal/transport/handlers"
	AuthHandler "main/internal/transport/handlers/auth"
	ChatHandler "main/internal/transport/handlers/chat"
	MessageHandler "main/internal/transport/handlers/message"
	UserHandler "main/internal/transport/handlers/user"
	"main/internal/transport/ws"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
)

const (
	envLocal = "local"
	envProd  = "prod"
	envDev   = "dev"
)

func main() {
	cfg := config.MustLoadConfig()
	secretKey := config.MySecretKey()

	logger := setupLogger(cfg.Env)

	logger.Info("Starting application", slog.String("env", cfg.Env))
	addr := net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port))

	router := chi.NewRouter()

	NewClaims, err := claims.NewClaims(secretKey)
	if err != nil {
		logger.Error("failed to create JWT claims", slog.String("error", err.Error()))
		return
	}

	dbConn, err := psql.NewDBPool(cfg.DatabaseDSN())
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		return
	}
	defer dbConn.Close()

	if err := dbConn.Ping(context.Background()); err != nil {
		logger.Error("failed to ping database", slog.String("error", err.Error()))
		return
	}
	redis, err := rdb.NewRedisClient(context.Background(), cfg.Redis.Addr, cfg.Redis.Password)
	if err != nil {
		logger.Error("failed to connect to redis", slog.String("error", err.Error()))
		return
	}
	defer redis.Close()

	NewCache := rdb.NewCache(redis)

	authRepo := auth.NewAuthRepository(dbConn, logger)
	userRepo := user.NewUserRepository(dbConn, logger)
	chatRepo := chat.NewChatRepository(dbConn, logger)
	msgRepo := msg.NewMessageRepository(dbConn, logger)
	jwtService := srvAuth.NewTokenService()
	NewJWTFacade := srvAuth.NewJWTFacade(NewClaims, NewCache)

	userService := srvUser.NewUserService(userRepo, logger)
	authService := srvAuth.NewAuthService(authRepo, jwtService, NewCache)
	chatService := srvChat.NewChatService(userRepo, chatRepo, logger)
	messageService := srvMessage.NewMessageService(chatRepo, msgRepo, logger)

	wsManager := ws.NewManager(logger)

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	logger.Info("Connected to database successfully")

	router.Use(middleware.RequestID)
	router.Use(mwMiddleware.New(logger))
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

	userHandler := UserHandler.NewUserHandler(userService, authService, upgrader, NewJWTFacade, logger)
	authHandler := AuthHandler.NewAuthHandler(authService, NewJWTFacade, logger)
	chatHandler := ChatHandler.NewChatHandler(messageService, chatService, logger, NewJWTFacade)
	messageHandler := MessageHandler.NewMessageHandler(messageService, chatService, logger, wsManager, NewJWTFacade)

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
