package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware "main/internal/delivery/http/middleware_auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockJWTManager struct {
	mock.Mock
}

func (m *MockJWTManager) Parse(accessToken string) (int64, error) {
	args := m.Called(accessToken)
	if args.Get(0) == nil {
		return 0, args.Error(1)
	}
	return args.Get(0).(int64), nil
}

func (m *MockJWTManager) Exists(ctx context.Context, token string) (bool, error) {

	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return false, args.Error(1)
	}

	return true, nil
}

func TestJWTAuth(t *testing.T) {
	tests := []struct {
		name           string
		headerName     string
		headerValue    string
		tokenString    string
		mockBehavior   func(m *MockJWTManager)
		expectedCode   int
		expectedUserID int64
	}{
		{
			name:        "Success (Valid Token)",
			headerName:  "Authorization",
			headerValue: "Bearer valid_token",
			tokenString: "valid_token",
			mockBehavior: func(m *MockJWTManager) {
				m.On("Exists", mock.Anything, "valid_token").Return(false, nil)
				m.On("Parse", "valid_token").Return(int64(10), nil)
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
			tokenString: "banned_token",
			mockBehavior: func(m *MockJWTManager) {
				m.On("Exists", mock.Anything, "banned_token").Return(true, nil)
			},
			expectedCode: 401,
		},
		{
			name:        "Invalid Token Signature (Parse Error)",
			headerName:  "Authorization",
			headerValue: "Bearer bad_sign_token",
			tokenString: "bad_sign_token",
			mockBehavior: func(m *MockJWTManager) {
				m.On("Exists", mock.Anything, "bad_sign_token").Return(false, nil)
				m.On("Parse", "bad_sign_token").Return(int64(0), errors.New("invalid signature"))
			},
			expectedCode: 401,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockJWTManager)
			tt.mockBehavior(mockManager)

			middleware := middleware.JWTAuth(mockManager)

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				if tt.expectedUserID != 0 {
					userID := r.Context().Value("userID")
					assert.Equal(t, tt.expectedUserID, userID, "UserID not found in context")
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/", nil)
			if tt.headerName != "" {
				req.Header.Set(tt.headerName, tt.headerValue)
			}
			w := httptest.NewRecorder()

			middleware(nextHandler).ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			mockManager.AssertExpectations(t)
		})
	}
}
