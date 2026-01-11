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

	msg "main/internal/database/mongo"
	psql "main/internal/database/postgres"
	auth "main/internal/database/postgres/auth_repo"
	chat "main/internal/database/postgres/chat_repo"
	user "main/internal/database/postgres/user_repo"
	rdb "main/internal/database/redis"
	kafka "main/internal/infrastructure/kafka"
	claims "main/internal/pkg/jwt"
	mwMiddleware "main/internal/server/middleware"
	httpHandler "main/internal/transport/handlers"
	AuthHandler "main/internal/transport/handlers/auth"
	ChatHandler "main/internal/transport/handlers/chat"
	MessageHandler "main/internal/transport/handlers/message"
	UserHandler "main/internal/transport/handlers/user"
	"main/internal/transport/ws"
	srvAuth "main/internal/usecase/auth"
	srvChat "main/internal/usecase/chat"
	eventHandler "main/internal/usecase/event"
	srvMessage "main/internal/usecase/message"
	srvUser "main/internal/usecase/user"

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

	deletedConsumerHandler := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		"msg_deleted_topic",
		"chat_group",
	)

	//--------------Databases Connections-----------------
	postgres, err := psql.NewDBPool(cfg.DatabaseDSN())
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		return
	}
	defer postgres.Close()
	if err := postgres.Ping(context.Background()); err != nil {
		logger.Error("failed to ping database", slog.String("error", err.Error()))
		return
	}

	mongoClient, err := msg.NewMongoClient(context.Background(), cfg.MongoURI())
	if err != nil {
		logger.Error("failed to connect to mongo", slog.String("error", err.Error()))
		return
	}
	defer func() {
		if err := mongoClient.Client().Disconnect(context.Background()); err != nil {
			logger.Error("failed to disconnect mongo client", slog.String("error", err.Error()))
		}
	}()
	redis, err := rdb.NewRedisClient(context.Background(), cfg.Redis)
	if err != nil {
		logger.Error("failed to connect to redis", slog.String("error", err.Error()))
		return
	}
	defer redis.Close()

	//-----------------------Repositories---------------------------
	NewCache := rdb.NewCache(redis)
	authRepo := auth.NewAuthRepository(postgres, logger)
	userRepo := user.NewUserRepository(postgres)
	chatRepo := chat.NewChatRepository(postgres, logger)
	msgRepo := msg.NewMessageRepository(mongoClient, logger)
	jwtService := srvAuth.NewTokenService()
	NewJWTFacade := srvAuth.NewJWTFacade(NewClaims, NewCache)

	//-----------------------Kafka-------------------------------
	event := eventHandler.NewEventHandlers(chatRepo, msgRepo)
	deletedProducer := kafka.NewProducer(cfg.Kafka.Brokers, "msg_deleted_topic")
	createdProducer := kafka.NewProducer(cfg.Kafka.Brokers, "msg_created_topic")
	defer deletedProducer.Close()
	defer createdProducer.Close()
	deletedConsumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		"msg_deleted_topic",
		"chat_group",
		event.HandleMessageDeleted,
		logger,
	)
	createdConsumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		"msg_created_topic",
		"chat_group",
		event.HandleMessageCreated,
		logger,
	)
	consumerManager := kafka.NewConsumerManager([]*kafka.Consumer{deletedConsumer, createdConsumer})
	go func() {
		if err := consumerManager.StartAll(context.Background()); err != nil {
			logger.Error("Kafka consumers stopped with error", slog.String("error", err.Error()))
		}
	}()

	//-----------------------Services-------------------------------
	userService := srvUser.NewUserService(userRepo, logger)
	authService := srvAuth.NewAuthService(authRepo, jwtService, NewCache)
	chatService := srvChat.NewChatService(userRepo, chatRepo, logger)
	messageService := srvMessage.NewMessageService(chatRepo, msgRepo, logger)

	//-----------------------HTTP Server-------------------------------

	wsManager := ws.NewManager(logger)
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

	//-----------------------Handlers-------------------------------
	userHandler := UserHandler.NewUserHandler(userService, NewJWTFacade, logger)
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
