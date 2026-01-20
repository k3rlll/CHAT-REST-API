package auth_repo_test

import (
	"context"
	auth "main/internal/database/postgres/auth_repo"
	dbtest "main/internal/database/postgres/repositoryTest"
	dom "main/internal/domain/entity"
	"main/pkg/customerrors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func truncateTables(ctx context.Context, t *testing.T, pool *pgxpool.Pool, tables ...string) {
	for _, table := range tables {
		_, err := pool.Exec(ctx, "TRUNCATE TABLE "+table+" CASCADE")
		if err != nil {
			t.Fatalf("failed to truncate table %s: %v", table, err)
		}
	}
}

func TestGetCredentialsByUsername(t *testing.T) {
	pool, teardown := dbtest.SetupTestDB(t)
	defer teardown()

	repo := auth.NewAuthRepository(pool, nil)
	ctx := context.Background()

	_, err := pool.Exec(ctx, "INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, $4)",
		1, "testuser", "test@example.com", "hash123")
	assert.NoError(t, err)

	tests := []struct {
		name          string
		username      string
		expectedID    int64
		expectedPass  string
		expectError   bool
		expectedError error
	}{
		{
			name:          "Valid user",
			username:      "testuser",
			expectedID:    1,
			expectedPass:  "hash123",
			expectError:   false,
			expectedError: nil,
		},
		{
			name:          "User not found",
			username:      "ghost",
			expectedID:    0,
			expectedPass:  "",
			expectError:   true,
			expectedError: customerrors.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetCredentialsByUsername(ctx, tt.username)

			if tt.expectError {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, user.ID)
				assert.Equal(t, tt.expectedPass, user.Password)
				assert.Equal(t, tt.username, user.Username)
			}
		})
	}
}

func TestSaveRefreshToken(t *testing.T) {
	pool, teardown := dbtest.SetupTestDB(t)
	defer teardown()

	repo := auth.NewAuthRepository(pool, nil)
	ctx := context.Background()

	_, err := pool.Exec(ctx, "INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, $4)",
		1, "testuser", "testuser@example.com", "hashedpassword")
	assert.NoError(t, err)

	tests := []struct {
		name          string
		token         dom.RefreshToken
		expectError   bool
		expectedError error
	}{
		{
			name: "Save valid token",
			token: dom.RefreshToken{
				UserID:    1,
				Token:     "refresh_token_123",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			expectError:   false,
			expectedError: nil,
		},
		{
			name: "Foreign key violation",
			token: dom.RefreshToken{
				UserID:    999,
				Token:     "bad_token",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			expectError:   true,
			expectedError: customerrors.ErrDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			truncateTables(ctx, t, pool, "refresh_tokens")

			err := repo.SaveRefreshToken(ctx, tt.token)

			if tt.expectError {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)

				var savedToken string
				err = pool.QueryRow(ctx, "SELECT refresh_token FROM refresh_tokens WHERE user_id=$1", tt.token.UserID).Scan(&savedToken)
				assert.NoError(t, err)
				assert.Equal(t, tt.token.Token, savedToken)
			}
		})
	}
}

func TestGetRefreshToken(t *testing.T) {
	pool, teardown := dbtest.SetupTestDB(t)
	defer teardown()

	repo := auth.NewAuthRepository(pool, nil)
	ctx := context.Background()

	tests := []struct {
		name          string
		setup         func()
		userID        int64
		expectedToken string
		expectError   bool
		expectedError error
	}{
		{
			name: "Get valid token",
			setup: func() {
				_, _ = pool.Exec(ctx, "INSERT INTO users (id, username, email, password_hash) VALUES (1, 'u1', 'e1', 'p1')")
				_, _ = pool.Exec(ctx, "INSERT INTO refresh_tokens (user_id, refresh_token, created_at, expires_at) VALUES (1, 'valid_token', NOW(), NOW() + INTERVAL '1 day')")
			},
			userID:        1,
			expectedToken: "valid_token",
			expectError:   false,
			expectedError: nil,
		},
		{
			name: "Token not found",
			setup: func() {
				_, _ = pool.Exec(ctx, "INSERT INTO users (id, username, email, password_hash) VALUES (2, 'u2', 'e2', 'p2')")
			},
			userID:        2,
			expectedToken: "",
			expectError:   true,
			expectedError: customerrors.ErrRefreshTokenNotFound,
		},
		{
			name: "Token expired",
			setup: func() {
				_, _ = pool.Exec(ctx, "INSERT INTO users (id, username, email, password_hash) VALUES (3, 'u3', 'e3', 'p3')")
				_, _ = pool.Exec(ctx, "INSERT INTO refresh_tokens (user_id, refresh_token, created_at, expires_at) VALUES (3, 'expired_token', NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 day')")
			},
			userID:        3,
			expectedToken: "",
			expectError:   true,
			expectedError: customerrors.ErrRefreshTokenExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			truncateTables(ctx, t, pool, "users", "refresh_tokens")

			if tt.setup != nil {
				tt.setup()
			}

			token, err := repo.GetRefreshToken(ctx, tt.userID)

			if tt.expectError {
				assert.ErrorIs(t, err, tt.expectedError)
				if tt.expectedError == customerrors.ErrRefreshTokenExpired {
					var count int
					err := pool.QueryRow(ctx, "SELECT count(*) FROM refresh_tokens WHERE user_id=$1", tt.userID).Scan(&count)
					assert.NoError(t, err)
					assert.Equal(t, 0, count)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}

func TestDeleteRefreshToken(t *testing.T) {
	pool, teardown := dbtest.SetupTestDB(t)
	defer teardown()

	repo := auth.NewAuthRepository(pool, nil)
	ctx := context.Background()

	_, err := pool.Exec(ctx, "INSERT INTO users (id, username, email, password_hash) VALUES (1, 'u1', 'e1', 'p1')")
	assert.NoError(t, err)
	_, err = pool.Exec(ctx, "INSERT INTO refresh_tokens (user_id, refresh_token, created_at, expires_at) VALUES (1, 'del_token', NOW(), NOW() + INTERVAL '1 day')")
	assert.NoError(t, err)

	err = repo.DeleteRefreshToken(ctx, 1)
	assert.NoError(t, err)

	var count int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM refresh_tokens WHERE user_id=$1", 1).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}
