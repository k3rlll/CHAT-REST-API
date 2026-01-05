package message_repo_test

import (
	"context"
	"fmt"
	"main/internal/database/postgres/message_repo"
	"main/internal/database/postgres/repositoryTest"
	dom "main/internal/domain/entity"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateMessage(t *testing.T) {
	testMessage := dom.Message{
		Id:             1,
		ChatID:         1,
		SenderID:       1,
		SenderUsername: "testuser",
		Text:           "Hello, World!",
		CreatedAt:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	ctx := context.Background()
	pool, teardown := repositoryTest.SetupTestDB(t)
	defer teardown()
	repo := message_repo.NewMessageRepository(pool)
	if _, err := pool.Exec(ctx, "insert into chats (id, title) values (1, 'Test Chat')"); err != nil {
		t.Fatalf("failed to insert chat: %v", err)
	}
	_, err := pool.Exec(ctx, "insert into users (id, username, email, password_hash) values ($1, $2, $3, $4)",
		testMessage.Id, testMessage.SenderUsername, "testuser@example.com", "hashedpassword")
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	if _, err := pool.Exec(ctx,
		"insert into chat_members (chat_id, user_id) values (1, 1)"); err != nil {
		t.Fatalf("failed to insert chat member: %v", err)
	}

	tests := []struct {
		name              string
		chatID            int64
		userID            int64
		testMessage       dom.Message
		expectErr         bool
		expectedErrString string
	}{
		{
			name:        "Successful Message Creation",
			chatID:      1,
			userID:      1,
			testMessage: testMessage,
			expectErr:   false,
		},
		{
			name:              "Fail to create message with non-existing chat",
			chatID:            999,
			userID:            1,
			testMessage:       testMessage,
			expectErr:         true,
			expectedErrString: "violates foreign key constraint",
		},
		{
			name:              "Fail to create message with non-existing user",
			chatID:            1,
			userID:            999,
			testMessage:       testMessage,
			expectErr:         true,
			expectedErrString: "violates foreign key constraint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			createdMsg, err := repo.Create(ctx, tt.chatID, tt.userID, tt.testMessage.SenderUsername, tt.testMessage.Text)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				assert.Contains(t, err.Error(), tt.expectedErrString)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				assert.Equal(t, tt.testMessage.Text, createdMsg.Text)
				assert.Equal(t, tt.chatID, createdMsg.ChatID)
				assert.Equal(t, tt.userID, createdMsg.SenderID)

				assert.NotZero(t, createdMsg.Id)
				assert.NotZero(t, createdMsg.CreatedAt)
			}
		})
	}
}

func TestListByChat(t *testing.T) {

	pool, teardown := repositoryTest.SetupTestDB(t)
	defer teardown()
	repo := message_repo.NewMessageRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, "INSERT INTO users (id, username, email, password_hash) VALUES (1, 'tester', 'test@mail.com', 'hash')")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "INSERT INTO chats (id, title) VALUES (10, 'Target Chat'), (99, 'Noise Chat')")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "INSERT INTO chat_members (chat_id, user_id) VALUES (10, 1), (99, 1)")
	require.NoError(t, err)

	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := 1; i <= 20; i++ {
		msgTime := baseTime.Add(time.Duration(i) * time.Minute)
		_, err := pool.Exec(ctx,
			"INSERT INTO messages (id, chat_id, sender_id, sender_username, text, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
			i, 10, 1, "tester", fmt.Sprintf("msg %d", i), msgTime)
		require.NoError(t, err)
	}

	_, err = pool.Exec(ctx, "INSERT INTO messages (id, chat_id, sender_id, sender_username, text) VALUES (100, 99, 1, 'tester', 'noise msg')")
	require.NoError(t, err)

	tests := []struct {
		name            string
		chatID          int64
		limit           int
		lastMessage     int
		expectedCount   int
		expectedFirstID int64
		expectedLastID  int64
	}{
		{
			name:            "First Page: Get latest 5 messages",
			chatID:          10,
			limit:           5,
			lastMessage:     math.MaxInt32,
			expectedCount:   5,
			expectedFirstID: 20,
			expectedLastID:  16,
		},
		{
			name:            "Second Page: Get next 5 messages after ID 16",
			chatID:          10,
			limit:           5,
			lastMessage:     16,
			expectedCount:   5,
			expectedFirstID: 15,
			expectedLastID:  11,
		},
		{
			name:            "Last Page: Remaining items",
			chatID:          10,
			limit:           50,
			lastMessage:     6,
			expectedCount:   5,
			expectedFirstID: 5,
			expectedLastID:  1,
		},
		{
			name:            "Empty Result: Wrong Chat ID",
			chatID:          999,
			limit:           10,
			lastMessage:     math.MaxInt32,
			expectedCount:   0,
			expectedFirstID: 0,
			expectedLastID:  0,
		},
		{
			name:            "Noise Chat: Should verify isolation",
			chatID:          99,
			limit:           10,
			lastMessage:     math.MaxInt32,
			expectedCount:   1,
			expectedFirstID: 100,
			expectedLastID:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs, err := repo.ListByChat(ctx, tt.chatID, tt.limit, tt.lastMessage)

			assert.NoError(t, err)
			assert.Len(t, msgs, tt.expectedCount)

			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedFirstID, msgs[0].Id, "First message ID mismatch")
				assert.Equal(t, tt.expectedLastID, msgs[len(msgs)-1].Id, "Last message ID mismatch")

				for i := 0; i < len(msgs)-1; i++ {
					assert.True(t, msgs[i].Id > msgs[i+1].Id, "Messages are not sorted DESC")
				}
			}
		})
	}
}
