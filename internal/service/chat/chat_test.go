package chat

import (
	"context"
	dom "main/internal/domain/chat"

	"github.com/stretchr/testify/mock"
)

type MockChatRepo struct {
	mock.Mock
}

func (m *MockChatRepo) GetChatDetails(ctx context.Context, chatID int64) (dom.Chat, error) {
	args := m.Called(ctx, chatID)
	if args.Get(0) == nil {
		return dom.Chat{}, args.Error(1)
	}
	return args.Get(0).(dom.Chat), args.Error(1)
}
func (m *MockChatRepo) ListOfChats(ctx context.Context, userID int64) ([]dom.Chat, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dom.Chat), args.Error(1)
}

func (m *MockChatRepo) CreateChat(ctx context.Context, title string, isPrivate bool, members []int64) (int64, error) {
	args := m.Called(ctx, title, isPrivate, members)
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
