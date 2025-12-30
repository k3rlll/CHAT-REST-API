package mock

import (
	context "context"
	entity "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	service "main/internal/service/message"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSendMessage(t *testing.T) {
	testChatID := int64(1)
	testUserID := int64(1)
	senderUsername := "user1"
	text := "Hello, world!"
	mockMessage := entity.Message{
		Id:             1,
		Text:           text,
		CreatedAt:      time.Now(),
		ChatID:         testChatID,
		SenderID:       testUserID,
		SenderUsername: senderUsername,
	}

	tests := []struct {
		name           string
		behavior       func(m *MockMessageInterface, c *MockChatInterface)
		expectedResult entity.Message
		expectedError  error
		isErr          bool
	}{
		{
			name: "Successful message sending",
			behavior: func(m *MockMessageInterface, c *MockChatInterface) {
				gomock.InOrder(
					c.EXPECT().CheckIsMemberOfChat(gomock.Any(), testChatID, testUserID).Return(true, nil),
					m.EXPECT().Create(gomock.Any(), testChatID, testUserID, senderUsername, text).Return(mockMessage, nil),
				)
			},
			expectedResult: mockMessage,
			expectedError:  nil,
			isErr:          false,
		},
		{
			name: "User not member of chat",
			behavior: func(m *MockMessageInterface, c *MockChatInterface) {
				c.EXPECT().CheckIsMemberOfChat(gomock.Any(), testChatID, testUserID).Return(false, nil)
			},
			expectedResult: entity.Message{},
			expectedError:  customerrors.ErrUserNotMemberOfChat,
			isErr:          true,
		},
		{
			name: "Database error on membership check",
			behavior: func(m *MockMessageInterface, c *MockChatInterface) {
				c.EXPECT().CheckIsMemberOfChat(gomock.Any(), testChatID, testUserID).Return(false, customerrors.ErrDatabase)
			},
			expectedResult: entity.Message{},
			expectedError:  customerrors.ErrDatabase,
			isErr:          true,
		},
		{
			name: "Database error on message creation",
			behavior: func(m *MockMessageInterface, c *MockChatInterface) {
				gomock.InOrder(
					c.EXPECT().CheckIsMemberOfChat(gomock.Any(), testChatID, testUserID).Return(true, nil),
					m.EXPECT().Create(gomock.Any(), testChatID, testUserID, senderUsername, text).Return(entity.Message{}, customerrors.ErrDatabase),
				)
			},
			expectedResult: entity.Message{},
			expectedError:  customerrors.ErrDatabase,
			isErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockMessageInterface := NewMockMessageInterface(ctrl)
			mockChatInterface := NewMockChatInterface(ctrl)

			if tt.behavior != nil {
				tt.behavior(mockMessageInterface, mockChatInterface)
			}
			messageService := service.NewMessageService(mockChatInterface, mockMessageInterface, nil)

			result, err := messageService.SendMessage(
				context.TODO(),
				testChatID,
				testUserID,
				senderUsername,
				text,
			)
			if tt.isErr {
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.expectedResult, result)
				}
			}
		})
	}
}

func TestDeleteMessage(t *testing.T) {
	testMessageID := int64(1)
	tests := []struct {
		name          string
		behavior      func(m *MockMessageInterface)
		expectedError error
		isErr         bool
	}{
		{
			name: "Successful message deletion",
			behavior: func(m *MockMessageInterface) {
				gomock.InOrder(
					m.EXPECT().CheckMessageExists(gomock.Any(), testMessageID).Return(true, nil),
					m.EXPECT().DeleteMessage(gomock.Any(), testMessageID).Return(nil),
				)
			},
			expectedError: nil,
			isErr:         false,
		},
		{
			name: "Message does not exist",
			behavior: func(m *MockMessageInterface) {
				m.EXPECT().CheckMessageExists(gomock.Any(), testMessageID).Return(false, nil)
			},
			expectedError: customerrors.ErrMessageDoesNotExists,
			isErr:         true,
		},
		{
			name: "Database error on existence check",
			behavior: func(m *MockMessageInterface) {
				m.EXPECT().CheckMessageExists(gomock.Any(), testMessageID).Return(false, customerrors.ErrDatabase)
			},
			expectedError: customerrors.ErrDatabase,
			isErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockMessageInterface := NewMockMessageInterface(ctrl)
			if tt.behavior != nil {
				tt.behavior(mockMessageInterface)
			}
			messageService := service.NewMessageService(nil, mockMessageInterface, nil)

			err := messageService.DeleteMessage(context.TODO(), testMessageID)
			if tt.isErr {
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestEditMessage(t *testing.T) {
	testMessageID := int64(1)
	newText := "Updated message text"
	tests := []struct {
		name          string
		behavior      func(m *MockMessageInterface)
		expectedError error
		isErr         bool
	}{
		{
			name: "Successful message edit",
			behavior: func(m *MockMessageInterface) {
				gomock.InOrder(
					m.EXPECT().CheckMessageExists(gomock.Any(), testMessageID).Return(true, nil),
					m.EXPECT().EditMessage(gomock.Any(), testMessageID, newText).Return(nil),
				)
			},
			expectedError: nil,
			isErr:         false,
		},
		{
			name: "Message does not exist",
			behavior: func(m *MockMessageInterface) {
				m.EXPECT().CheckMessageExists(gomock.Any(), testMessageID).Return(false, nil)
			},
			expectedError: customerrors.ErrMessageDoesNotExists,
			isErr:         true,
		},
		{
			name: "Database error on existence check",
			behavior: func(m *MockMessageInterface) {
				m.EXPECT().CheckMessageExists(gomock.Any(), testMessageID).Return(false, customerrors.ErrDatabase)
			},
			expectedError: customerrors.ErrDatabase,
			isErr:         true,
		},
		{
			name: "Database error on message edit",
			behavior: func(m *MockMessageInterface) {
				gomock.InOrder(
					m.EXPECT().CheckMessageExists(gomock.Any(), testMessageID).Return(true, nil),
					m.EXPECT().EditMessage(gomock.Any(), testMessageID, newText).Return(customerrors.ErrDatabase),
				)
			},
			expectedError: customerrors.ErrDatabase,
			isErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockMessageInterface := NewMockMessageInterface(ctrl)
			if tt.behavior != nil {
				tt.behavior(mockMessageInterface)
			}
			messageService := service.NewMessageService(nil, mockMessageInterface, nil)
			err := messageService.EditMessage(context.TODO(), testMessageID, newText)
			if tt.isErr {
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}
