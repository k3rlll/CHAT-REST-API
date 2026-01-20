package user_repo_test

import (
	"context"
	repositoryT "main/internal/database/postgres/repositoryTest"
	"main/internal/database/postgres/user_repo"
	"main/pkg/customerrors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterUser(t *testing.T) {
	pool, teardown := repositoryT.SetupTestDB(t)
	defer teardown()

	repo := user_repo.NewUserRepository(pool)
	ctx := context.Background()

	seedUser := func(username, email string) {
		_, err := pool.Exec(ctx,
			"INSERT INTO users (username, email, password_hash) VALUES ($1, $2, 'hash')",
			username, email)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		username      string
		email         string
		passwordHash  string
		setupDB       func()
		expectedError error
	}{
		{
			name:          "Success: Register new user",
			username:      "new_user",
			email:         "new@example.com",
			passwordHash:  "secret_hash",
			setupDB:       func() {},
			expectedError: nil,
		},
		{
			name:         "Fail: Username already exists",
			username:     "busy_user",
			email:        "unique@example.com",
			passwordHash: "123",
			setupDB: func() {
				seedUser("busy_user", "other@example.com")
			},
			expectedError: customerrors.ErrUsernameAlreadyExists,
		},
		{
			name:         "Fail: Email already exists",
			username:     "unique_user",
			email:        "busy@example.com",
			passwordHash: "123",
			setupDB: func() {

				seedUser("other_user", "busy@example.com")
			},
			expectedError: customerrors.ErrEmailAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE users RESTART IDENTITY CASCADE")
			require.NoError(t, err)
			if tt.setupDB != nil {
				tt.setupDB()
			}

			createdUser, err := repo.RegisterUser(ctx, tt.username, tt.email, tt.passwordHash)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)

				assert.Equal(t, tt.username, createdUser.Username)
				assert.Equal(t, tt.email, createdUser.Email)

				var count int
				err = pool.QueryRow(ctx,
					"SELECT count(*) FROM users WHERE username=$1 AND email=$2",
					tt.username, tt.email).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count, "User should be found in database")
			}
		})
	}
}
func TestSearchUser(t *testing.T) {

	pool, teardown := repositoryT.SetupTestDB(t)
	defer teardown()

	repo := user_repo.NewUserRepository(pool)
	ctx := context.Background()

	usersData := []string{"Alice", "Alicia", "Bob", "Charlie"}

	for _, username := range usersData {
		_, err := pool.Exec(ctx,
			"INSERT INTO users (username, email, password_hash) VALUES ($1, $2, 'hash')",
			username, username+"@example.com")
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "Exact match ",
			query:         "alice",
			expectedCount: 1,
			expectedNames: []string{"Alice"},
		},
		{
			name:          "Partial match (Beginning)",
			query:         "Ali",
			expectedCount: 2,
			expectedNames: []string{"Alice", "Alicia"},
		},
		{
			name:          "Partial match (Middle)",
			query:         "li",
			expectedCount: 3,
			expectedNames: []string{"Alice", "Alicia", "Charlie"},
		},
		{
			name:          "No match",
			query:         "Zorro",
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name:          "Empty query (Matches ALL usually)",
			query:         "",
			expectedCount: 4,
			expectedNames: []string{"Alice", "Alicia", "Bob", "Charlie"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			users, err := repo.SearchUser(ctx, tt.query)

			require.NoError(t, err)
			assert.Len(t, users, tt.expectedCount)

			foundNames := make(map[string]bool)
			for _, u := range users {
				foundNames[u.Username] = true
			}

			for _, expectedName := range tt.expectedNames {
				assert.True(t, foundNames[expectedName], "Expected to find user: %s", expectedName)
			}
		})
	}
}
