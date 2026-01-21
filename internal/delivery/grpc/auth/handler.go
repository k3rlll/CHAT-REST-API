package auth

import (
	"context"
	"log/slog"
	dom "main/internal/domain/entity"
	auth_gen "main/pkg/proto/gen/auth/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	auth_gen.UnimplementedAuthServiceServer
	authUsecase AuthService
	log         *slog.Logger
}

type AuthService interface {
	LoginUser(ctx context.Context, username, password string) (accessToken string, err error)
	LogoutUser(ctx context.Context, accessToken string) (bool, error)
	RegisterUser(ctx context.Context, username, email, password string) (dom.User, error)
}

func NewAuthHandler(authUsecase AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		authUsecase: authUsecase,
		log:         logger,
	}
}

func (h *AuthHandler) Login(ctx context.Context, req *auth_gen.LoginRequest) (*auth_gen.LoginResponse, error) {
	accessToken, err := h.authUsecase.LoginUser(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		h.log.Error("could not login user", "error", err, "email", req.GetEmail())
		return nil, status.Errorf(codes.Internal, "could not login user: %v", err)
	}
	h.log.Info("user logged in", "email", req.GetEmail())

	return &auth_gen.LoginResponse{
		Token: accessToken,
	}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *auth_gen.LogoutRequest) (*auth_gen.LogoutResponse, error) {
	success, err := h.authUsecase.LogoutUser(ctx, req.GetToken())
	if err != nil {
		h.log.Error("could not logout user", "error", err)
		return nil, status.Errorf(codes.Internal, "could not logout user: %v", err)
	}
	if !success {
		h.log.Error("logout failed", "token", req.GetToken())
		return nil, status.Errorf(codes.Internal, "could not logout user")
	}
	h.log.Info("user logged out", "token", req.GetToken())
	return &auth_gen.LogoutResponse{}, nil
}

func (h *AuthHandler) Register(ctx context.Context, req *auth_gen.RegisterRequest) (*auth_gen.RegisterResponse, error) {
	user, err := h.authUsecase.RegisterUser(ctx, req.GetEmail(), req.GetPassword(), req.GetUsername())
	if err != nil {
		h.log.Error("could not register user", "error", err)
		return nil, status.Errorf(codes.Internal, "could not register user: %v", err)
	}
	h.log.Info("user registered", "user_id", user.ID, "email", req.GetEmail())
	return &auth_gen.RegisterResponse{
		UserId: user.ID,
	}, nil
}
