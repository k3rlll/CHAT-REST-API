package middleware_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	middleware "main/internal/delivery/http/middleware/auth"
	"main/internal/pkg/jwt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockJWTManager struct {
	mock.Mock
}

func (m *MockJWTManager) Parse(accessToken string) (*jwt.TokenClaims, error) {
	args := m.Called(accessToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwt.TokenClaims), args.Error(1)
}

func (m *MockJWTManager) Exists(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}

func TestJWTAuth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name           string
		headerName     string
		headerValue    string
		mockBehavior   func(m *MockJWTManager)
		expectedCode   int
		expectedUserID int64
	}{
		{
			name:        "Success (Valid Token)",
			headerName:  "Authorization",
			headerValue: "Bearer valid_token",
			mockBehavior: func(m *MockJWTManager) {
				m.On("Exists", mock.Anything, "valid_token").Return(false, nil)
				m.On("Parse", "valid_token").Return(&jwt.TokenClaims{UserID: 10, Exp: 1234567890}, nil)
			},
			expectedCode:   200,
			expectedUserID: 10,
		},
		{
			name:         "Missing Header",
			headerName:   "",
			headerValue:  "",
			mockBehavior: func(m *MockJWTManager) {},
			expectedCode: 401,
		},
		{
			name:         "Invalid Header Format",
			headerName:   "Authorization",
			headerValue:  "Basic 12345",
			mockBehavior: func(m *MockJWTManager) {},
			expectedCode: 401,
		},
		{
			name:        "Token Revoked (Banned)",
			headerName:  "Authorization",
			headerValue: "Bearer banned_token",
			mockBehavior: func(m *MockJWTManager) {
				m.On("Exists", mock.Anything, "banned_token").Return(true, nil)
			},
			expectedCode: 401,
		},
		{
			name:        "Redis Error",
			headerName:  "Authorization",
			headerValue: "Bearer valid_token",
			mockBehavior: func(m *MockJWTManager) {
				m.On("Exists", mock.Anything, "valid_token").Return(false, errors.New("redis error"))
			},
			expectedCode: 500,
		},
		{
			name:        "Invalid Token Signature (Parse Error)",
			headerName:  "Authorization",
			headerValue: "Bearer bad_sign_token",
			mockBehavior: func(m *MockJWTManager) {
				m.On("Exists", mock.Anything, "bad_sign_token").Return(false, nil)
				m.On("Parse", "bad_sign_token").Return(nil, errors.New("invalid signature"))
			},
			expectedCode: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockJWTManager)
			tt.mockBehavior(mockManager)

			mw := middleware.JWTAuth(mockManager, mockManager, logger)

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.expectedUserID != 0 {
					userID, ok := middleware.GetUserID(r.Context())
					assert.True(t, ok)
					assert.Equal(t, tt.expectedUserID, userID)
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/", nil)
			if tt.headerName != "" {
				req.Header.Set(tt.headerName, tt.headerValue)
			}
			w := httptest.NewRecorder()

			mw(nextHandler).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			mockManager.AssertExpectations(t)
		})
	}
}
