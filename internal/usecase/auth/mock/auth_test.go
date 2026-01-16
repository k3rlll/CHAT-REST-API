package mock_test

import (
	"context"
	"fmt"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	"main/internal/pkg/jwt"
	"main/internal/usecase/auth"
	mock "main/internal/usecase/auth/mock"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginUser(t *testing.T) {
	password := "securepassword"
	wrongPassword := "wrongpassword"
	accessToken := "right_access"
	refreshToken := "right_refresh"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	defaultTTL := 15 * time.Minute
	userID := int64(1)

	validUser := dom.User{
		ID:       userID,
		Username: "user",
		Password: string(hashedPassword),
	}

	tests := []struct {
		name               string
		ctx                context.Context
		username           string
		password           string
		setupMock          func(*mock.MockAuthRepository, *mock.MockTokenManager)
		expectAccessToken  string
		expectRefreshToken string
		expectError        error
	}{
		{
			name:     "Success login",
			ctx:      context.Background(),
			username: "user",
			password: password,
			setupMock: func(repo *mock.MockAuthRepository, token *mock.MockTokenManager) {
				gomock.InOrder(
					repo.EXPECT().
						GetCredentialsByUsername(gomock.Any(), "user").
						Return(validUser, nil),
					token.EXPECT().
						NewRefreshToken().
						Return(refreshToken, nil),
					token.EXPECT().
						NewAccessToken(userID, defaultTTL).
						Return(accessToken, nil),
					repo.EXPECT().
						SaveRefreshToken(gomock.Any(), gomock.Any()).
						Return(nil),
				)
			},
			expectAccessToken:  accessToken,
			expectRefreshToken: refreshToken,
			expectError:        nil,
		},
		{
			name:     "Invalid password",
			ctx:      context.Background(),
			username: "user",
			password: wrongPassword,
			setupMock: func(repo *mock.MockAuthRepository, token *mock.MockTokenManager) {
				repo.EXPECT().
					GetCredentialsByUsername(gomock.Any(), "user").
					Return(validUser, nil)
			},
			expectAccessToken:  "",
			expectRefreshToken: "",
			expectError:        customerrors.ErrInvalidInput,
		},
		{
			name:     "User not found",
			ctx:      context.Background(),
			username: "ghost",
			password: password,
			setupMock: func(repo *mock.MockAuthRepository, token *mock.MockTokenManager) {
				repo.EXPECT().
					GetCredentialsByUsername(gomock.Any(), "ghost").
					Return(dom.User{}, customerrors.ErrUserNotFound)
			},
			expectAccessToken:  "",
			expectRefreshToken: "",
			expectError:        customerrors.ErrUserNotFound,
		},
		{
			name:     "Database error",
			ctx:      context.Background(),
			username: "user",
			password: password,
			setupMock: func(repo *mock.MockAuthRepository, token *mock.MockTokenManager) {
				repo.EXPECT().
					GetCredentialsByUsername(gomock.Any(), "user").
					Return(dom.User{}, customerrors.ErrDatabase)
			},
			expectAccessToken:  "",
			expectRefreshToken: "",
			expectError:        customerrors.ErrDatabase,
		},
		{
			name:     "Token creation failed",
			ctx:      context.Background(),
			username: "user",
			password: password,
			setupMock: func(repo *mock.MockAuthRepository, token *mock.MockTokenManager) {
				gomock.InOrder(
					repo.EXPECT().GetCredentialsByUsername(gomock.Any(), "user").Return(validUser, nil),
					token.EXPECT().NewRefreshToken().Return("", customerrors.ErrTokenCreationFailed),
				)
			},
			expectAccessToken:  "",
			expectRefreshToken: "",
			expectError:        customerrors.ErrTokenCreationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mock.NewMockAuthRepository(ctrl)
			token := mock.NewMockTokenManager(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(repo, token)
			}

			s := auth.NewAuthService(repo, token, nil, defaultTTL)
			gotAccess, gotRefresh, err := s.LoginUser(tt.ctx, tt.username, tt.password)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectAccessToken, gotAccess)
				assert.Equal(t, tt.expectRefreshToken, gotRefresh.Token)
			}
		})
	}
}

func TestLogoutUser(t *testing.T) {
	accessToken := "valid_access_token"
	refreshToken := "valid_refresh_token"
	userID := int64(100)
	expTime := time.Now().Add(10 * time.Minute)

	tests := []struct {
		name        string
		ctx         context.Context
		access      string
		refresh     string
		setupMock   func(*mock.MockAuthRepository, *mock.MockTokenManager, *mock.MockTokenBlacklister)
		expectError error
	}{
		{
			name:    "Success logout",
			ctx:     context.Background(),
			access:  accessToken,
			refresh: refreshToken,
			setupMock: func(repo *mock.MockAuthRepository, token *mock.MockTokenManager, blacklist *mock.MockTokenBlacklister) {
				gomock.InOrder(
					token.EXPECT().
						Parse(accessToken).
						Return(&jwt.TokenClaims{UserID: userID, Exp: expTime.Unix()}, nil),
					blacklist.EXPECT().
						Set(gomock.Any(), accessToken, "blacklisted", gomock.Any()).
						Return(nil),
					repo.EXPECT().
						DeleteRefreshToken(gomock.Any(), userID).
						Return(nil),
				)
			},
			expectError: nil,
		},
		{
			name:    "Invalid access token",
			ctx:     context.Background(),
			access:  "invalid",
			refresh: refreshToken,
			setupMock: func(repo *mock.MockAuthRepository, token *mock.MockTokenManager, blacklist *mock.MockTokenBlacklister) {
				token.EXPECT().
					Parse("invalid").
					Return(nil, fmt.Errorf("invalid token"))
			},
			expectError: fmt.Errorf("invalid token"),
		},
		{
			name:    "Redis error",
			ctx:     context.Background(),
			access:  accessToken,
			refresh: refreshToken,
			setupMock: func(repo *mock.MockAuthRepository, token *mock.MockTokenManager, blacklist *mock.MockTokenBlacklister) {
				gomock.InOrder(
					token.EXPECT().
						Parse(accessToken).
						Return(&jwt.TokenClaims{UserID: userID, Exp: expTime.Unix()}, nil),
					blacklist.EXPECT().
						Set(gomock.Any(), accessToken, "blacklisted", gomock.Any()).
						Return(customerrors.ErrDatabase),
				)
			},
			expectError: customerrors.ErrDatabase,
		},
		{
			name:    "DB error",
			ctx:     context.Background(),
			access:  accessToken,
			refresh: refreshToken,
			setupMock: func(repo *mock.MockAuthRepository, token *mock.MockTokenManager, blacklist *mock.MockTokenBlacklister) {
				gomock.InOrder(
					token.EXPECT().
						Parse(accessToken).
						Return(&jwt.TokenClaims{UserID: userID, Exp: expTime.Unix()}, nil),
					blacklist.EXPECT().
						Set(gomock.Any(), accessToken, "blacklisted", gomock.Any()).
						Return(nil),
					repo.EXPECT().
						DeleteRefreshToken(gomock.Any(), userID).
						Return(customerrors.ErrDatabase),
				)
			},
			expectError: customerrors.ErrDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mock.NewMockAuthRepository(ctrl)
			token := mock.NewMockTokenManager(ctrl)
			blacklist := mock.NewMockTokenBlacklister(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(repo, token, blacklist)
			}

			s := auth.NewAuthService(repo, token, blacklist, 15*time.Minute)
			err := s.LogoutUser(tt.ctx, tt.access, tt.refresh)

			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
