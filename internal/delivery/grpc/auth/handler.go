package auth

import (
	"context"
	"log/slog"
	AuthUsecase "main/internal/usecase/auth"
	auth_gen "main/pkg/proto/gen/auth"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	auth_gen.UnimplementedAuthServiceServer
	authUsecase AuthUsecase.AuthService
	log         slog.Logger
}

func NewAuthHandler(authUsecase AuthUsecase.AuthService, logger slog.Logger) *AuthHandler {
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

func (h *AuthHandler) Logout(ctx context.Context, req *auth_gen.LogoutRequest) (*auth_gen.LogoutResponse, bool, error) {
	err := h.authUsecase.LogoutUser(ctx, req.GetToken())
	if err != nil {
		h.log.Error("could not logout user", "error", err)
		return nil, false, status.Errorf(codes.Internal, "could not logout user: %v", err)
	}
	h.log.Info("user logged out", "token", req.GetToken())
	return &auth_gen.LogoutResponse{}, true, nil
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
