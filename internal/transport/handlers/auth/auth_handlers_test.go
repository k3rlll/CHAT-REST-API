package handlers_test

import (
	"io"
	"log/slog"
	mocks "main/internal/service/auth"
	handlers "main/internal/transport/handlers/auth"
	"net/http"
	"testing"
)

func TestAuthHandler_LoginHandler(t *testing.T) {

	mockAuth := &mocks.AuthService{}
	mockTokenManager := &mocks.JWTManager{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		authSrv      handlers.AuthService
		tokenManager handlers.JWTManager
		logger       *slog.Logger
		// Named input parameters for target function.
		w http.ResponseWriter
		r *http.Request
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handlers.NewAuthHandler(tt.authSrv, tt.tokenManager, tt.logger)
			h.LoginHandler(tt.w, tt.r)
		})
	}
}
