package chat

import (
	"context"
	"errors"
	"io"
	"log/slog"
	domChat "main/internal/domain/chat"
	domMessage "main/internal/domain/message"
	"main/internal/pkg/customerrors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockChatRepo struct {
	mock.Mock
}

func (m *MockChatRepo) GetChatDetails(ctx context.Context, chatID int64) (domChat.Chat, error) {
	args := m.Called(ctx, chatID)
	if args.Get(0) == nil {
		return domChat.Chat{}, args.Error(1)
	}
	return args.Get(0).(domChat.Chat), args.Error(1)
}
func (m *MockChatRepo) ListOfChats(ctx context.Context, userID int64) ([]domChat.Chat, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domChat.Chat), args.Error(1)
}

func (m *MockChatRepo) CreateChat(ctx context.Context, title string, isPrivate bool, members []int64) (int64, error) {
	args := m.Called(ctx, title, isPrivate, members)
	if args.Get(0) == nil {
		return 0, args.Error(1)
	}
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockChatRepo) DeleteChat(ctx context.Context, chatID int64) error {
	args := m.Called(ctx, chatID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}
func (m *MockChatRepo) CheckIfChatExists(ctx context.Context, chatID int64) (bool, error) {
	args := m.Called(ctx, chatID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockChatRepo) CheckIsMemberOfChat(ctx context.Context, chatID int64, userID int64) (bool, error) {
	args := m.Called(ctx, chatID, userID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockChatRepo) OpenChat(ctx context.Context, chatID int64, userID int64) ([]domMessage.Message, error) {
	args := m.Called(ctx, chatID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domMessage.Message), args.Error(1)
}

func (m *MockChatRepo) UserInChat(ctx context.Context, chatID int64, userID int64) (bool, error) {
	args := m.Called(ctx, chatID, userID)
	return args.Get(0).(bool), args.Error(1)
}
func (m *MockChatRepo) AddMembers(ctx context.Context, chatID int64, members []int64) error {
	args := m.Called(ctx, chatID, members)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(0)
}

type MockUser struct {
	mock.Mock
}

func (m *MockUser) CheckUserExists(ctx context.Context, userID int64) bool {
	args := m.Called(ctx, userID)
	return args.Get(0).(bool)
}

func TestChatService_CreateChat(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	members := []int64{22, 33, 44}

	tests := []struct {
		name         string
		inputTitle   string
		inputPrivate bool
		inputMembers []int64
		mockBehavior func(r *MockChatRepo)
		expectedChat domChat.Chat
		expectError  bool
	}{
		{
			name:         "Successful creation",
			inputTitle:   "test",
			inputPrivate: true,
			inputMembers: members,
			mockBehavior: func(r *MockChatRepo) {
				r.On("CreateChat", mock.Anything, "test", true, members).
					Return(int64(777), nil)
			},
			expectedChat: domChat.Chat{
				Id:        777,
				Title:     "test",
				IsPrivate: true,
			},
			expectError: false,
		},
		{
			name:         "Empty title",
			inputTitle:   "",
			inputPrivate: true,
			inputMembers: members,
			mockBehavior: func(r *MockChatRepo) {
			},
			expectedChat: domChat.Chat{},
			expectError:  true,
		},
		{
			name:         "No member",
			inputTitle:   "test",
			inputPrivate: true,
			inputMembers: []int64{},
			mockBehavior: func(r *MockChatRepo) {},
			expectedChat: domChat.Chat{},
			expectError:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockChatRepo)
			tt.mockBehavior(mockRepo)
			chatService := NewChatService(nil, mockRepo, logger)
			chat, err := chatService.CreateChat(context.Background(), tt.inputPrivate, tt.inputTitle, tt.inputMembers)

			if tt.expectError {
				assert.Error(t, err)
				if tt.inputTitle == "" {
					assert.ErrorIs(t, err, customerrors.ErrInvalidInput)
				}
				if len(tt.inputMembers) == 0 {
					assert.ErrorIs(t, err, customerrors.ErrInvalidInput)
				}
				assert.Empty(t, chat)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedChat, chat)
				assert.Equal(t, int64(777), chat.Id)
			}

			mockRepo.AssertExpectations(t)

		})
	}

}

func TestChatService_DeleteChat(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	chatNotExists := errors.New("chat does not exist")
	randErr := errors.New("boom!")

	tests := []struct {
		name         string
		chatid       int64
		mockBehavior func(r *MockChatRepo)
		expectErr    bool
		expectedErr  error
	}{
		{
			name:   "Success",
			chatid: 1,
			mockBehavior: func(r *MockChatRepo) {
				r.On("CheckIfChatExists", mock.Anything, int64(1)).
					Return(true, nil)
				r.On("DeleteChat", mock.Anything, int64(1)).
					Return(nil)
			},
			expectErr:   false,
			expectedErr: nil,
		},
		{
			name:   "Chat does not exist",
			chatid: 1,
			mockBehavior: func(r *MockChatRepo) {
				r.On("CheckIfChatExists", mock.Anything, int64(1)).
					Return(false, chatNotExists)
			},
			expectErr:   true,
			expectedErr: chatNotExists,
		},
		{
			name:   "failed to check existence of the chat",
			chatid: 1,
			mockBehavior: func(r *MockChatRepo) {
				r.On("CheckIfChatExists", mock.Anything, int64(1)).
					Return(false, randErr)
			},
			expectErr:   true,
			expectedErr: randErr,
		},
		{
			name:   "failed to delete chat",
			chatid: 1,
			mockBehavior: func(r *MockChatRepo) {
				r.On("CheckIfChatExists", mock.Anything, int64(1)).
					Return(true, nil)
				r.On("DeleteChat", mock.Anything, int64(1)).
					Return(randErr)
			},
			expectErr:   true,
			expectedErr: randErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockChatRepo)
			tt.mockBehavior(mockRepo)
			chatService := NewChatService(nil, mockRepo, logger)
			err := chatService.DeleteChat(context.Background(), tt.chatid)

			if tt.expectErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			mock.AssertExpectationsForObjects(t)
		})
	}
}
