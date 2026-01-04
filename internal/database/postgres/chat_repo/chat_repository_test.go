package chat_repo_test

import (
	"context"
	"main/internal/database/postgres/chat_repo"
	dbtest "main/internal/database/postgres/repositoryTest"
	"main/internal/pkg/customerrors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateChat(t *testing.T) {
	ctx := context.Background()
	pool, teardown := dbtest.SetupTestDB(t)
	defer teardown()
	_, err := pool.Exec(ctx,
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, $4)", 1, "testuser1", "testuser1@example.com", "hashedpassword")
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}
	_, err = pool.Exec(ctx,
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, $4)", 2, "testuser2", "testuser2@example.com", "hashedpassword")
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}
	_, err = pool.Exec(ctx,
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, $4)", 3, "testuser3", "testuser3@example.com", "hashedpassword")
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}
	one_membersID := []int64{1}
	two_membersID := []int64{1, 2}
	three_membersID := []int64{1, 2, 3}

	chatRepo := chat_repo.NewChatRepository(pool, nil)
	tests := []struct {
		name          string
		title         string
		isPrivate     bool
		membersID     []int64
		expectErr     bool
		expectedError error
	}{
		{
			name:          "Create chat with one member and no title",
			title:         "",
			isPrivate:     true,
			membersID:     one_membersID,
			expectErr:     false,
			expectedError: nil,
		},
		{
			name:          "Create chat with two members and no title",
			title:         "",
			isPrivate:     true,
			membersID:     two_membersID,
			expectErr:     false,
			expectedError: nil,
		},
		{
			name:          "Create chat with three members and no title",
			title:         "",
			isPrivate:     false,
			membersID:     three_membersID,
			expectErr:     false,
			expectedError: nil,
		},
		{
			name:          "Create chat with title and two members",
			title:         "Group Chat",
			isPrivate:     false,
			membersID:     two_membersID,
			expectErr:     false,
			expectedError: nil,
		},
		{
			name:          "Create chat with empty membersID",
			title:         "Empty Members Chat",
			isPrivate:     true,
			membersID:     []int64{},
			expectErr:     true,
			expectedError: customerrors.ErrDatabase,
		},
		{
			name:          "Break the constraint by adding non-existing user",
			title:         "Invalid Members Chat",
			isPrivate:     true,
			membersID:     []int64{999},
			expectErr:     true,
			expectedError: customerrors.ErrDatabase,
		},
		{
			name:          "Break the constraint by adding no members",
			title:         "No Members Chat",
			isPrivate:     true,
			membersID:     []int64{},
			expectErr:     true,
			expectedError: customerrors.ErrDatabase,
		},
		{
			name:          "Break the constraint by adding duplicate members",
			title:         "Duplicate Members Chat",
			isPrivate:     true,
			membersID:     []int64{1, 1},
			expectErr:     true,
			expectedError: customerrors.ErrDatabase,
		},
		{
			name:          "Break the constraint by adding to long title",
			title:         "This title is way too long to be accepted by the database constraints imposed on the chat title field",
			isPrivate:     true,
			membersID:     one_membersID,
			expectErr:     true,
			expectedError: customerrors.ErrDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE TABLE chats CASCADE")
			if err != nil {
				t.Fatalf("failed to truncate chats table: %v", err)
			}
			_, err = chatRepo.CreateChat(ctx, tt.title, tt.isPrivate, tt.membersID)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				assert.NoError(t, err)
			}
		})
	}

}
