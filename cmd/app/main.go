package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	config "main/internal/config"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	msg "main/internal/database/mongo"
	psql "main/internal/database/postgres"
	auth "main/internal/database/postgres/auth_repo"
	chat "main/internal/database/postgres/chat_repo"
	user "main/internal/database/postgres/user_repo"
	rdb "main/internal/database/redis"
	authRPC "main/internal/delivery/grpc/auth"
	interceptor "main/internal/delivery/grpc/middleware"
	httpHandler "main/internal/delivery/http"
	ChatHandler "main/internal/delivery/http/chat"
	MessageHandler "main/internal/delivery/http/message"
	"main/internal/delivery/http/middleware/metrics"
	UserHandler "main/internal/delivery/http/user"
	"main/internal/delivery/ws"
	kafka "main/internal/infrastructure/kafka"
	srvAuth "main/internal/usecase/auth"
	srvChat "main/internal/usecase/chat"
	eventHandler "main/internal/usecase/event"
	srvMessage "main/internal/usecase/message"
	srvUser "main/internal/usecase/user"
	claims "main/pkg/jwt"
	pb "main/pkg/proto/gen/auth/v1"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

const (
	envLocal = "local"
	envProd  = "prod"
	envDev   = "dev"
)

type CombinedTokenManager struct {
	*claims.Manager
	*rdb.Cache
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found")
	}
	cfg := config.MustLoadConfig()
	secretKey := config.MySecretKey()

	logger := setupLogger(cfg.Env)
	logger.Info("Starting application", slog.String("env", cfg.Env))
	addr := net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port))
	router := chi.NewRouter()

	//-----------------------Metrics Router---------------------------
	metricsRouter := chi.NewRouter()
	metricsRouter.Handle("/metrics", promhttp.Handler())

	//-----------------------JWT Claims---------------------------
	jwtManager, err := claims.NewManager(secretKey)
	if err != nil {
		logger.Error("failed to create JWT manager", slog.String("error", err.Error()))
		return
	}

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

	tokenController := &CombinedTokenManager{
		Manager: jwtManager,
		Cache:   NewCache,
	}

	//-----------------------Kafka-------------------------------
	event := eventHandler.NewEventHandlers(chatRepo, msgRepo)
	producer := kafka.NewProducer(cfg.Kafka.Brokers)
	defer producer.Close()
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
	authService := srvAuth.NewAuthService(authRepo, userRepo, jwtManager, NewCache, cfg.Auth.TokenTTL)
	chatService := srvChat.NewChatService(userRepo, chatRepo, logger)
	messageService := srvMessage.NewMessageService(chatRepo, msgRepo, producer, logger)

	//-----------------------HTTP Server-------------------------------

	wsManager := ws.NewManager(logger)
	logger.Info("Connected to database successfully")

	router.Use(middleware.RequestID)
	router.Use(metrics.PrometheusMiddleware)
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
	userHandler := UserHandler.NewUserHandler(userService, tokenController, logger)
	chatHandler := ChatHandler.NewChatHandler(messageService, chatService, logger, tokenController)
	messageHandler := MessageHandler.NewMessageHandler(messageService, chatService, logger, wsManager, tokenController)
	authRpcHandler := authRPC.NewAuthHandler(authService, logger)

	HTTP := httpHandler.NewHTTPHandler(userHandler, chatHandler, messageHandler, logger)
	HTTP.RegisterRoutes(router)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverParams := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.Timeout,
		WriteTimeout: cfg.Server.Timeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
	metricsParams := &http.Server{
		Addr:    net.JoinHostPort(cfg.Metrics.Host, strconv.Itoa(cfg.Metrics.Port)),
		Handler: metricsRouter,
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			interceptor.AuthInterceptor(jwtManager),
		),
	)
	pb.RegisterAuthServiceServer(grpcServer, authRpcHandler)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		logger.Info("Metrics server is starting", slog.String("addr", metricsParams.Addr))
		if err := metricsParams.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})
	g.Go(func() error {
		lis, err := net.Listen("tcp", net.JoinHostPort(cfg.Grpc.Host, strconv.Itoa(cfg.Grpc.Port)))
		if err != nil {
			return err
		}
		logger.Info("gRPC server is starting", slog.String("addr", lis.Addr().String()))
		if err := grpcServer.Serve(lis); err != nil {
			return err
		}
		return nil
	})

	g.Go(func() error {
		logger.Info("HTTP server is starting", slog.String("addr", serverParams.Addr))
		if err := serverParams.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})
	// --- Shutdown Orchestrator ---
	g.Go(func() error {
		<-gCtx.Done()
		logger.Info("shutting down servers...")

		shutDownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(3)

		go func() {
			defer wg.Done()
			if err := serverParams.Shutdown(shutDownCtx); err != nil {
				logger.Error("HTTP server shutdown failed", slog.String("error", err.Error()))
			}
		}()

		go func() {
			defer wg.Done()
			if err := metricsParams.Shutdown(shutDownCtx); err != nil {
				logger.Error("Metrics server shutdown failed", slog.String("error", err.Error()))
			}
		}()

		go func() {
			defer wg.Done()
			grpcServer.GracefulStop()
		}()

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
			logger.Info("All servers stopped gracefully")
		case <-shutDownCtx.Done():
			logger.Warn("Shutdown timeout exceeded, forcing stop")
			grpcServer.Stop()
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		if !errors.Is(err, context.Canceled) {
			logger.Error("Application stopped with error", slog.Any("err", err))
			os.Exit(1)
		}
	}
	logger.Info("Application exited properly")

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envLocal, envDev:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	default:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}
