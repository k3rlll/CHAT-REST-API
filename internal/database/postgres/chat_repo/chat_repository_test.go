package chat_repo_test

import (
	"context"
	"main/internal/database/postgres/chat_repo"
	dbtest "main/internal/database/postgres/repositoryTest"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	"testing"
	"time"

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
			name:          "Break the constraint by adding non-existing user",
			title:         "Invalid Members Chat",
			isPrivate:     true,
			membersID:     []int64{999},
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
			name: "Break the constraint by adding to long title",
			title: func() string {
				s := ""
				for i := 0; i < 300; i++ {
					s += "a"
				}
				return s
			}(),
			isPrivate:     true,
			membersID:     one_membersID,
			expectErr:     true,
			expectedError: customerrors.ErrDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, "TRUNCATE chats cascade")
			if err != nil {
				t.Fatalf("failed to truncate chats table: %v", err)
			}
			_, err = pool.Exec(ctx, "TRUNCATE chat_members cascade")
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

func TestOpenChat(t *testing.T) {
	ctx := context.Background()
	pool, teardown := dbtest.SetupTestDB(t)
	defer teardown()
	repo := chat_repo.NewChatRepository(pool, nil)

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
	_, err = pool.Exec(ctx,
		"INSERT INTO chats (id, title, is_private) VALUES ($1, $2, $3)", 1, "Test Chat", false)
	if err != nil {
		t.Fatalf("failed to insert test chat: %v", err)
	}
	_, err = pool.Exec(ctx,
		"INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)", 1, 1)
	if err != nil {
		t.Fatalf("failed to insert chat member: %v", err)
	}
	_, err = pool.Exec(ctx,
		"INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)", 1, 2)
	if err != nil {
		t.Fatalf("failed to insert chat member: %v", err)
	}
	_, err = pool.Exec(ctx,
		"INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)", 1, 3)
	if err != nil {
		t.Fatalf("failed to insert chat member: %v", err)
	}

	testMessages := []dom.Message{
		{Id: 1, ChatID: 1, SenderID: 1, Text: "Hello", CreatedAt: time.Date(2024, time.January, 1, 10, 0, 0, 0, time.UTC)},
		{Id: 2, ChatID: 1, SenderID: 2, Text: "Hi there!", CreatedAt: time.Date(2024, time.January, 1, 10, 1, 0, 0, time.UTC)},
		{Id: 3, ChatID: 1, SenderID: 1, Text: "How are you?", CreatedAt: time.Date(2024, time.January, 1, 10, 2, 0, 0, time.UTC)},
	}
	for _, msg := range testMessages {
		_, err = pool.Exec(ctx,
			"INSERT INTO messages (id, chat_id, sender_id, text, created_at) VALUES ($1, $2, $3, $4, $5)",
			msg.Id, msg.ChatID, msg.SenderID, msg.Text, msg.CreatedAt)
		if err != nil {
			t.Fatalf("failed to insert message: %v", err)
		}
	}
	tests := []struct {
		name          string
		chatID        int64
		userID        int64
		expectedMsgs  []dom.Message
		expectErr     bool
		expectedError error
	}{
		{
			name:          "Open chat as member",
			chatID:        1,
			userID:        2,
			expectedMsgs:  testMessages,
			expectErr:     false,
			expectedError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := repo.OpenChat(ctx, tt.chatID, tt.userID)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error but got none")
					assert.ErrorIs(t, err, tt.expectedError)
				} else {
					t.Fatalf("unexpected error: %v", err)
					assert.Equal(t, tt.expectedMsgs, messages)
				}
			}
		})
	}
}
