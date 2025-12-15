package auth

import (
	"context"
	dom "main/internal/domain/user"
	customerrors "main/internal/pkg/customerrors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ------- MockUserRepo implements the UserRepo interface for testing purposes.
type MockAuthRepo struct {
	mock.Mock
}

func (m *MockAuthRepo) GetPasswordHash(ctx context.Context, refreshToken string, userID int64, password string) (dom.User, error) {
	args := m.Called(ctx, refreshToken, userID, password)
	if args.Get(0) == nil {
		return dom.User{}, args.Error(1)
	}

	return args.Get(0).(dom.User), args.Error(1)
}

func (m *MockAuthRepo) SaveRefreshToken(ctx context.Context, userID int64, refreshToken string) error {
	args := m.Called(ctx, userID, refreshToken)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAuthRepo) DeleteRefreshToken(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

func (m *MockAuthRepo) CheckPasswordHash(password, hash string) bool {
	args := m.Called(password, hash)
	if args.Get(0) == nil {
		return false
	}
	return args.Bool(0)
}

// ------- MockRedis implements the Rdb interface for testing purposes.
type MockSet struct {
	mock.Mock
}

func (m *MockSet) Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error {
	args := m.Called(ctx, key, value, ttlSeconds)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

// ------- MockToken implements the Token interface for testing purposes.
type MockToken struct {
	mock.Mock
}

func (m *MockToken) NewAccessToken(userID int64, ttl time.Duration) (string, error) {
	args := m.Called(userID, ttl)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

func (m *MockToken) NewRefreshToken() (string, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

// Testing code will be here
func TestAuthService_LoginUser(t *testing.T) {
	testUser := dom.User{
		ID:       1,
		Username: "testuser",
		Password: "hashedpassword",
	}

	tests := []struct {
		name               string
		userID             int64
		password           string
		mockBehavior       func(repo *MockAuthRepo, token *MockToken, redis *MockSet)
		expectAccessToken  string
		expectRefreshToken string
		expectError        bool
	}{
		{
			name:     "Successful login",
			userID:   1,
			password: "password",
			mockBehavior: func(repo *MockAuthRepo, token *MockToken, redis *MockSet) {
				repo.On("GetPasswordHash", mock.Anything, mock.Anything, int64(1), "password").Return(testUser, nil)
				repo.On("CheckPasswordHash", "password", "hashedpassword").Return(true)
				token.On("NewAccessToken", int64(1), mock.Anything).Return("access_token", nil)
				token.On("NewRefreshToken").Return("refresh_token", nil)
				repo.On("SaveRefreshToken", mock.Anything, int64(1), "refresh_token").Return(nil)
			},
			expectAccessToken:  "access_token",
			expectRefreshToken: "refresh_token",
			expectError:        false},
		{
			name:     "User not found",
			userID:   1,
			password: "any",
			mockBehavior: func(repo *MockAuthRepo, token *MockToken, redis *MockSet) {
				repo.On("GetPasswordHash", mock.Anything, mock.Anything, int64(1), "any").Return(testUser, customerrors.ErrUserNotFound)

			},
			expectAccessToken:  "",
			expectRefreshToken: "",
			expectError:        true},
		{
			name:     "Invalid password",
			userID:   1,
			password: "wrongpassword",
			mockBehavior: func(repo *MockAuthRepo, token *MockToken, redis *MockSet) {
				repo.On("GetPasswordHash", mock.Anything, mock.Anything, int64(1), "wrongpassword").Return(testUser, nil)
				repo.On("CheckPasswordHash", "wrongpassword", "hashedpassword").Return(false)
			},
			expectAccessToken:  "",
			expectRefreshToken: "",
			expectError:        true},
		{
			name:     "Failed to generate access token",
			userID:   1,
			password: "password",
			mockBehavior: func(repo *MockAuthRepo, token *MockToken, redis *MockSet) {
				repo.On("GetPasswordHash", mock.Anything, mock.Anything, int64(1), "password").Return(testUser, nil)
				repo.On("CheckPasswordHash", "password", "hashedpassword").Return(true)
				token.On("NewAccessToken", int64(1), mock.Anything).Return("", customerrors.ErrTokenCreationFailed)
			},
			expectAccessToken:  "",
			expectRefreshToken: "",
			expectError:        true},

		{
			name:     "Failed to save refresh token",
			userID:   1,
			password: "password",
			mockBehavior: func(repo *MockAuthRepo, token *MockToken, redis *MockSet) {
				repo.On("GetPasswordHash", mock.Anything, mock.Anything, int64(1), "password").Return(testUser, nil)
				repo.On("CheckPasswordHash", "password", "hashedpassword").Return(true)
				token.On("NewAccessToken", int64(1), mock.Anything).Return("access_token", nil)
				token.On("NewRefreshToken").Return("refresh_token", nil)
				repo.On("SaveRefreshToken", mock.Anything, int64(1), "refresh_token").Return(customerrors.ErrFailedToSaveToken)
			},
			expectAccessToken:  "",
			expectRefreshToken: "",
			expectError:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAuthRepo)
			mockToken := new(MockToken)
			mockRedis := new(MockSet)

			tt.mockBehavior(mockRepo, mockToken, mockRedis)
			authService := NewAuthService(mockRepo, mockToken, mockRedis)
			accessToken, refreshToken, err := authService.LoginUser(context.Background(), tt.userID, tt.password)

			if tt.expectError {
				if err == nil {
					assert.Error(t, err, "expected an error but got none")
					assert.Empty(t, accessToken, "expected empty access token")
					assert.Empty(t, refreshToken, "expected empty refresh token")
				} else {
					assert.NotEmpty(t, err.Error(), "expected an error message")
					assert.NoError(t, err, "did not expect an error")
					assert.Equal(t, tt.expectAccessToken, accessToken, "access token does not match")
					assert.Equal(t, tt.expectRefreshToken, refreshToken, "refresh token does not match")
				}

				mockRepo.AssertExpectations(t)
				mockToken.AssertExpectations(t)
				mockRedis.AssertExpectations(t)
			}
		})
	}
}
