package mock

import (
	context "context"
	"fmt"
	entity "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	"main/internal/usecase/auth"
	"testing"

	gomock "go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

// TODO: recover tests after fixing the issues
func TestLoginUsers(t *testing.T) {
	password := "securepassword"
	accessToken := "right_access"
	refreshToken := "right_refresh"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)

	tests := []struct {
		name               string
		Behavior           func(*MockAuthRepository, *MockToken)
		ctx                context.Context
		userID             int64
		username           string
		password           string
		hashedPassword     string
		expectPassword     string
		accessToken        string
		refreshToken       string
		expectAccessToken  string
		expectRefreshToken string
		wantUser           entity.User
		wantErr            bool
		expectErr          error
	}{
		{
			name:           "ok",
			ctx:            context.Background(),
			userID:         1,
			password:       password,
			hashedPassword: string(hashedPassword),
			accessToken:    accessToken,
			refreshToken:   refreshToken,
			Behavior: func(repo *MockAuthRepository, token *MockToken) {
				gomock.InOrder(
					repo.EXPECT().
						GetPasswordHash(gomock.Any(), gomock.Any(), password).Return(entity.User{ID: 1, Password: string(hashedPassword)}, nil),
					token.EXPECT().
						NewAccessToken(int64(1), gomock.Any()).Return(accessToken, nil),
					token.EXPECT().
						NewRefreshToken().Return(refreshToken, nil),
					repo.EXPECT().
						SaveRefreshToken(gomock.Any(), refreshToken).Return(nil),
				)
			},
			wantUser:           entity.User{ID: 1, Username: "testuser"},
			wantErr:            false,
			expectErr:          nil,
			expectPassword:     password,
			expectAccessToken:  accessToken,
			expectRefreshToken: refreshToken,
		}, {
			name:           "invalid password",
			ctx:            context.Background(),
			userID:         1,
			password:       "wrongpassword",
			hashedPassword: string(hashedPassword),
			accessToken:    accessToken,
			refreshToken:   refreshToken,
			Behavior: func(repo *MockAuthRepository, token *MockToken) {
				gomock.InOrder(
					repo.EXPECT().
						GetPasswordHash(gomock.Any(), int64(1), "wrongpassword").Return(entity.User{ID: 1, Password: string(hashedPassword)}, nil),
				)
			},
			wantUser:           entity.User{},
			wantErr:            true,
			expectErr:          customerrors.ErrInvalidInput,
			expectPassword:     "wrongpassword",
			expectAccessToken:  "",
			expectRefreshToken: "",
		},
		{
			name:           "repo error",
			ctx:            context.Background(),
			userID:         1,
			password:       password,
			hashedPassword: string(hashedPassword),
			accessToken:    accessToken,
			refreshToken:   refreshToken,
			Behavior: func(repo *MockAuthRepository, token *MockToken) {
				gomock.InOrder(
					repo.EXPECT().
						GetPasswordHash(gomock.Any(), int64(1), password).Return(entity.User{}, customerrors.ErrDatabase),
				)
			},
			wantUser:           entity.User{},
			wantErr:            true,
			expectErr:          customerrors.ErrDatabase,
			expectPassword:     password,
			expectAccessToken:  "",
			expectRefreshToken: "",
		},
		{
			name:           "token creation error",
			ctx:            context.Background(),
			userID:         1,
			password:       password,
			hashedPassword: string(hashedPassword),
			accessToken:    accessToken,
			refreshToken:   refreshToken,
			Behavior: func(repo *MockAuthRepository, token *MockToken) {
				gomock.InOrder(
					repo.EXPECT().
						GetPasswordHash(gomock.Any(), int64(1), password).Return(entity.User{ID: 1, Password: string(hashedPassword)}, nil),
					token.EXPECT().
						NewAccessToken(int64(1), gomock.Any()).Return("", customerrors.ErrTokenCreationFailed),
				)
			},
			wantUser:           entity.User{},
			wantErr:            true,
			expectErr:          customerrors.ErrTokenCreationFailed,
			expectPassword:     password,
			expectAccessToken:  "",
			expectRefreshToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMockAuthRepository(ctrl)
			token := NewMockToken(ctrl)

			if tt.Behavior != nil {
				tt.Behavior(repo, token)
			}
			s := auth.NewAuthService(repo, token, nil)
			gotAccessToken, gotRefreshToken, err := s.LoginUser(tt.ctx, tt.userID, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoginUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err != tt.expectErr {
				t.Errorf("LoginUser() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if gotAccessToken != tt.expectAccessToken {
				t.Errorf("LoginUser() gotAccessToken = %v, expectAccessToken %v", gotAccessToken, tt.expectAccessToken)
			}
			if gotRefreshToken.Token != tt.expectRefreshToken {
				t.Errorf("LoginUser() gotRefreshToken = %v, expectRefreshToken %v", gotRefreshToken, tt.expectRefreshToken)
			}
		})
	}
}

func TestLogoutUser(t *testing.T) {
	accessToken := "right_access"
	tests := []struct {
		name        string
		Behavior    func(*MockAuthRepository, *MockSetInterface)
		ctx         context.Context
		userID      int64
		accessToken string
		wantErr     bool
		expectErr   error
	}{
		{
			name:        "ok",
			ctx:         context.Background(),
			userID:      1,
			accessToken: accessToken,
			Behavior: func(repo *MockAuthRepository, redis *MockSetInterface) {
				gomock.InOrder(
					redis.EXPECT().
						Set(gomock.Any(), accessToken, "blacklist", 60*15).Return(nil),
					repo.EXPECT().
						DeleteRefreshToken(gomock.Any(), int64(1)).Return(nil),
				)
			},
			wantErr:   false,
			expectErr: nil,
		},
		{
			name:        "redis set error",
			ctx:         context.Background(),
			userID:      1,
			accessToken: accessToken,
			Behavior: func(repo *MockAuthRepository, redis *MockSetInterface) {
				gomock.InOrder(
					redis.EXPECT().
						Set(gomock.Any(), accessToken, "blacklist", 60*15).Return(customerrors.ErrDatabase),
				)
			},
			wantErr:   true,
			expectErr: fmt.Errorf("failed to blacklist access token: %w", customerrors.ErrDatabase),
		},
		{
			name:        "repo delete error",
			ctx:         context.Background(),
			userID:      1,
			accessToken: accessToken,
			Behavior: func(repo *MockAuthRepository, redis *MockSetInterface) {
				gomock.InOrder(
					redis.EXPECT().
						Set(gomock.Any(), accessToken, "blacklist", 60*15).Return(nil),
					repo.EXPECT().
						DeleteRefreshToken(gomock.Any(), int64(1)).Return(customerrors.ErrDatabase),
				)
			},
			wantErr:   true,
			expectErr: fmt.Errorf("failed to delete refresh token: %w", customerrors.ErrDatabase),
		},
		{
			name:        "invalid userID",
			ctx:         context.Background(),
			userID:      -1,
			accessToken: accessToken,
			Behavior: func(repo *MockAuthRepository, redis *MockSetInterface) {

			},
			wantErr:   true,
			expectErr: fmt.Errorf("invalid userID"),
		},
		{
			name:        "empty access token",
			ctx:         context.Background(),
			userID:      1,
			accessToken: "",
			Behavior: func(repo *MockAuthRepository, redis *MockSetInterface) {
			},
			wantErr:   true,
			expectErr: fmt.Errorf("access token is empty"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMockAuthRepository(ctrl)
			redis := NewMockSetInterface(ctrl)
			if tt.Behavior != nil {
				tt.Behavior(repo, redis)
			}
			s := auth.NewAuthService(repo, nil, redis)
			err := s.LogoutUser(tt.ctx, tt.userID, tt.accessToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("LogoutUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && err.Error() != tt.expectErr.Error() {
				t.Errorf("LogoutUser() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
		})
	}
}
