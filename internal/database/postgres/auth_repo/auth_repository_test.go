package auth_repo_test

import (
	"context"
	auth "main/internal/database/postgres/auth_repo"
	dbtest "main/internal/database/postgres/repositoryTest"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSaveRefreshToken(t *testing.T) {
	pool, teardown := dbtest.SetupTestDB(t)
	defer teardown()

	repo := auth.NewAuthRepository(pool, nil)
	ctx := context.Background()

	_, err := pool.Exec(ctx, "INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, $4)", 1, "testuser", "testuser@example.com", "hashedpassword")
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	testToken := dom.RefreshToken{
		UserID:    1,
		Token:     "sample_refresh_token",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	tests := []struct {
		name          string
		testToken     dom.RefreshToken
		expectError   bool
		expectedError error
	}{
		{
			name:          "Save valid refresh token",
			testToken:     testToken,
			expectError:   false,
			expectedError: nil,
		},
		{
			name: "Save refresh token with missing user ID",
			testToken: dom.RefreshToken{
				UserID: 0,
			},
			expectError:   true,
			expectedError: customerrors.ErrDatabase,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE refresh_tokens")
			if err != nil {
				t.Fatalf("failed to truncate table: %v", err)
			}
			err = repo.SaveRefreshToken(ctx, tt.testToken)

			if tt.expectError {
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)

				var savedToken string
				err = pool.QueryRow(ctx, "SELECT refresh_token FROM refresh_tokens WHERE user_id=$1", tt.testToken.UserID).Scan(&savedToken)
				assert.NoError(t, err)
				assert.Equal(t, tt.testToken.Token, savedToken)
			}
		})
	}

}

func TestGetRefreshToken(t *testing.T) {
	pool, teardown := dbtest.SetupTestDB(t)
	defer teardown()

	repo := auth.NewAuthRepository(pool, nil)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, $4)", 1, "testuser", "testuser@example.com", "hashedpassword")
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}
	_, err = pool.Exec(ctx,
		"INSERT INTO refresh_tokens (user_id, refresh_token, created_at, expires_at) VALUES ($1, $2, $3, $4)",
		1, "valid_token", time.Now(), time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("failed to insert refresh token: %v", err)
	}
	tests := []struct {
		name          string
		userID        int64
		expectedToken string
		expectError   bool
		expectedError error
	}{
		{
			name:          "Get valid refresh token",
			userID:        1,
			expectedToken: "valid_token",
			expectError:   false,
			expectedError: nil,
		},
		{
			name:          "Get non-existent refresh token",
			userID:        1,
			expectedToken: "",
			expectError:   true,
			expectedError: customerrors.ErrRefreshTokenNotFound,
		},
		{
			name:          "Get expired refresh token",
			userID:        1,
			expectedToken: "",
			expectError:   true,
			expectedError: customerrors.ErrRefreshTokenExpired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE refresh_tokens")
			if err != nil {
				t.Fatalf("failed to truncate table: %v", err)
			}
			if tt.name == "Get expired refresh token" {
				_, err = pool.Exec(ctx,
					"INSERT INTO refresh_tokens (user_id, refresh_token, created_at, expires_at) VALUES ($1, $2, $3, $4)",
					tt.userID, "expired_token", time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour))
			} else if tt.name == "Get non-existent refresh token" {
				// Do not insert any token
			} else {
				_, err = pool.Exec(ctx,
					"INSERT INTO refresh_tokens (user_id, refresh_token, created_at, expires_at) VALUES ($1, $2, $3, $4)",
					tt.userID, tt.expectedToken, time.Now(), time.Now().Add(24*time.Hour))
			}
			if err != nil {
				t.Fatalf("failed to setup test data: %v", err)
			}
			token, err := repo.GetRefreshToken(ctx, tt.userID)

			if tt.expectError {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}

func TestGetByEmail(t *testing.T) {
	pool, teardown := dbtest.SetupTestDB(t)
	defer teardown()

	repo := auth.NewAuthRepository(pool, nil)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, $4)", 1, "testuser", "testuser@example.com", "hashedpassword")
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	expectedUser := dom.User{
		ID:       1,
		Username: "testuser",
		Password: "hashedpassword",
	}

	tests := []struct {
		name          string
		email         string
		expectedUser  dom.User
		expectError   bool
		expectedError error
	}{
		{
			name:          "Get user by valid email",
			email:         "testuser@example.com",
			expectedUser:  expectedUser,
			expectError:   false,
			expectedError: nil,
		},
		{
			name:          "Get user by non-existent email",
			email:         "wrongemail@example.com",
			expectedUser:  dom.User{},
			expectError:   true,
			expectedError: customerrors.ErrUserNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetByEmail(ctx, tt.email)
			if tt.expectError {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUser, user)
			}
		})
	}
}
