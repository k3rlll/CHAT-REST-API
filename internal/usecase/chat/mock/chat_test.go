package mock_test

import (
	context "context"
	"fmt"
	dom "main/internal/domain/entity"
	entity "main/internal/domain/entity"
	service "main/internal/usecase/chat"
	"main/internal/usecase/chat/mock"
	"main/pkg/customerrors"
	"testing"

	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func TestCreateChat(t *testing.T) {

	title := "Test Chat"
	members := []int64{1, 2, 3}

	tests := []struct {
		name          string
		title         string
		isPrivate     bool
		members       []int64
		mockBehavior  func(chatRepo *mock.MockChatRepositoryInterface)
		isErr         bool
		expectedChat  dom.Chat
		expectedError error
	}{
		{
			name:      "Successful chat creation",
			title:     title,
			isPrivate: false,
			members:   members,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {
				chatRepo.EXPECT().CreateChat(gomock.Any(), title, false, members).Return(int64(1), nil)
			},
			expectedChat: dom.Chat{
				ID:        1,
				Title:     title,
				IsPrivate: false,
			},
			expectedError: nil,
			isErr:         false,
		},
		{
			name:      "Chat creation with empty title",
			title:     "",
			isPrivate: false,
			members:   members,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {

			},
			expectedChat:  dom.Chat{},
			expectedError: customerrors.ErrInvalidInput,
			isErr:         true,
		},
		{
			name:      "Chat creation with no members",
			title:     title,
			isPrivate: false,
			members:   []int64{},
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {

			},
			expectedChat:  dom.Chat{},
			expectedError: customerrors.ErrInvalidInput,
			isErr:         true,
		},
		{
			name:      "Repository error during chat creation",
			title:     title,
			isPrivate: false,
			members:   members,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {
				chatRepo.EXPECT().CreateChat(gomock.Any(), title, false, members).Return(int64(0), customerrors.ErrDatabase)
			},
			expectedChat:  dom.Chat{},
			expectedError: customerrors.ErrDatabase,
			isErr:         true,
		},
		{
			name:      "Chat creation with title exceeding max length",
			title:     "This title is way too long to be accepted",
			isPrivate: false,
			members:   members,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {
			},
			expectedChat:  dom.Chat{},
			expectedError: customerrors.ErrInvalidInput,
			isErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockChatRepo := mock.NewMockChatRepositoryInterface(ctrl)

			if tt.mockBehavior != nil {
				tt.mockBehavior(mockChatRepo)
			}
			ChatService := service.NewChatService(nil, mockChatRepo, nil)
			chat, err := ChatService.CreateChat(context.Background(), tt.title, tt.isPrivate, tt.members)

			if !assert.Equal(t, tt.expectedChat, chat) {
				t.Errorf("expected chat: %v, got: %v", tt.expectedChat, chat)
			}
			if tt.isErr {
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}

		})

	}

}

func TestDeleteChat(t *testing.T) {
	chatID := int64(1)
	tests := []struct {
		name          string
		chatID        int64
		mockBehavior  func(chatRepo *mock.MockChatRepositoryInterface)
		expectedError error
		isErr         bool
	}{
		{
			name:   "Successful chat deletion",
			chatID: chatID,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {
				chatRepo.EXPECT().CheckIfChatExists(gomock.Any(), chatID).Return(true, nil)
				chatRepo.EXPECT().DeleteChat(gomock.Any(), chatID).Return(nil)
			},
			expectedError: nil,
			isErr:         false,
		},
		{
			name:   "Chat does not exist",
			chatID: chatID,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {
				chatRepo.EXPECT().CheckIfChatExists(gomock.Any(), chatID).Return(false, nil)
			},
			expectedError: customerrors.ErrNotFound,
			isErr:         true,
		},
		{
			name:   "Repository error during existence check",
			chatID: chatID,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {
				chatRepo.EXPECT().CheckIfChatExists(gomock.Any(), chatID).Return(false, customerrors.ErrDatabase)
			},
			expectedError: customerrors.ErrDatabase,
			isErr:         true,
		},
		{
			name:   "Repository error during chat deletion",
			chatID: chatID,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {
				chatRepo.EXPECT().CheckIfChatExists(gomock.Any(), chatID).Return(true, nil)
				chatRepo.EXPECT().DeleteChat(gomock.Any(), chatID).Return(customerrors.ErrDatabase)
			},
			expectedError: customerrors.ErrDatabase,
			isErr:         true,
		},
		{
			name:   "Invalid chat ID",
			chatID: -1,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {
			},
			expectedError: customerrors.ErrInvalidInput,
			isErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockChatRepo := mock.NewMockChatRepositoryInterface(ctrl)
			if tt.mockBehavior != nil {
				tt.mockBehavior(mockChatRepo)
			}
			ChatService := service.NewChatService(nil, mockChatRepo, nil)
			err := ChatService.DeleteChat(context.Background(), tt.chatID)
			if tt.isErr {
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}
		})

	}
}

func TestOpenChat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatRepo := mock.NewMockChatRepositoryInterface(ctrl)
	mockMsgRepo := mock.NewMockMessageRepositoryInterface(ctrl)

	now := "2024-06-15T12:00:00Z"
	anchorID := "651eb1234567890abcdef123"
	limit := int64(20)
	testChat := entity.Chat{ID: 1, Title: "Test Chat", MembersID: []int64{10, 20}}
	testMsgs := []entity.Message{{Text: "Hello from Mongo"}}

	tests := []struct {
		name       string
		chatID     int64
		userID     int64
		anchorTime string
		setup      func()
		wantChat   entity.Chat
		wantMsgs   []entity.Message
		wantErr    error
	}{
		{
			name:       "Success: Chat opened fully",
			chatID:     1,
			userID:     10,
			anchorTime: now,
			setup: func() {
				mockChatRepo.EXPECT().CheckIfChatExists(gomock.Any(), int64(1)).Return(true, nil)
				mockChatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), int64(1), int64(10)).Return(true, nil)
				mockChatRepo.EXPECT().GetChatDetails(gomock.Any(), int64(1)).Return(testChat, nil)
				mockMsgRepo.EXPECT().GetMessages(gomock.Any(), int64(1), now, anchorID, limit).Return(testMsgs, nil)
			},
			wantChat: entity.Chat{ID: 1, Title: "Test Chat", MembersID: []int64{10, 20}, MembersCount: 2},
			wantMsgs: testMsgs,
			wantErr:  nil,
		},
		{
			name:    "Fail: Invalid ChatID (Negative)",
			chatID:  -5,
			userID:  10,
			setup:   func() {},
			wantErr: customerrors.ErrInvalidInput,
		},
		{
			name:   "Fail: Chat does not exist",
			chatID: 99,
			userID: 10,
			setup: func() {
				mockChatRepo.EXPECT().CheckIfChatExists(gomock.Any(), int64(99)).Return(false, nil)
			},
			wantErr: customerrors.ErrNotFound,
		},
		{
			name:   "Fail: User is not a member",
			chatID: 1,
			userID: 999,
			setup: func() {
				mockChatRepo.EXPECT().CheckIfChatExists(gomock.Any(), int64(1)).Return(true, nil)
				mockChatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), int64(1), int64(999)).Return(false, nil)
			},
			wantErr: customerrors.ErrUserNotMemberOfChat,
		},
		{
			name:   "Fail: Database error on details",
			chatID: 1,
			userID: 10,
			setup: func() {
				mockChatRepo.EXPECT().CheckIfChatExists(gomock.Any(), int64(1)).Return(true, nil)
				mockChatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), int64(1), int64(10)).Return(true, nil)
				mockChatRepo.EXPECT().GetChatDetails(gomock.Any(), int64(1)).Return(entity.Chat{}, fmt.Errorf("sql error"))
			},
			wantErr: fmt.Errorf("sql error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.setup()
			service := &service.ChatService{
				Chat: mockChatRepo,
				Msg:  mockMsgRepo,
			}

			gotChat, gotMsgs, err := service.OpenChat(context.Background(), tt.chatID, tt.userID, tt.anchorTime, anchorID, limit)

			if tt.wantErr != nil {
				assert.Error(t, err)

				assert.Contains(t, err.Error(), tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantChat, gotChat)
				assert.Equal(t, tt.wantMsgs, gotMsgs)
			}
		})
	}
}

func TestAddMembers(t *testing.T) {
	chatID := int64(1)
	userID := int64(1)
	testMembers := []int64{2, 3}
	tests := []struct {
		name          string
		chatID        int64
		userID        int64
		members       []int64
		mockBehavior  func(chatRepo *mock.MockChatRepositoryInterface, userSvc *mock.MockUserInterface)
		expectedError error
		isErr         bool
	}{
		{
			name:    "Successful add members",
			chatID:  chatID,
			userID:  userID,
			members: testMembers,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface, userSvc *mock.MockUserInterface) {
				userSvc.EXPECT().CheckUserExists(gomock.Any(), userID).Return(true)
				chatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), chatID, userID).Return(true, nil)
				userSvc.EXPECT().CheckUserExists(gomock.Any(), testMembers[0]).Return(true)
				chatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), chatID, testMembers[0]).Return(false, nil)
				userSvc.EXPECT().CheckUserExists(gomock.Any(), testMembers[1]).Return(true)
				chatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), chatID, testMembers[1]).Return(false, nil)
				chatRepo.EXPECT().AddMembers(gomock.Any(), chatID, testMembers).Return(nil)
			},
			expectedError: nil,
			isErr:         false,
		},
		{
			name:    "User not found",
			chatID:  chatID,
			userID:  userID,
			members: testMembers,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface, userSvc *mock.MockUserInterface) {
				userSvc.EXPECT().CheckUserExists(gomock.Any(), userID).Return(false)
			},

			expectedError: customerrors.ErrUserNotFound,
			isErr:         true,
		},
		{
			name:    "User is not member of chat",
			chatID:  chatID,
			userID:  userID,
			members: testMembers,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface, userSvc *mock.MockUserInterface) {
				userSvc.EXPECT().CheckUserExists(gomock.Any(), userID).Return(true)
				chatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), chatID, userID).Return(false, nil)
			},
			expectedError: customerrors.ErrUserNotMemberOfChat,
			isErr:         true,
		},
		{
			name:    "New member not found",
			chatID:  chatID,
			userID:  userID,
			members: testMembers,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface, userSvc *mock.MockUserInterface) {
				userSvc.EXPECT().CheckUserExists(gomock.Any(), userID).Return(true)
				chatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), chatID, userID).Return(true, nil)
				userSvc.EXPECT().CheckUserExists(gomock.Any(), testMembers[0]).Return(false)
			},
			expectedError: customerrors.ErrUserNotFound,
			isErr:         true,
		},
		{
			name:    "New member already in chat",
			chatID:  chatID,
			userID:  userID,
			members: testMembers,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface, userSvc *mock.MockUserInterface) {
				userSvc.EXPECT().CheckUserExists(gomock.Any(), userID).Return(true)
				chatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), chatID, userID).Return(true, nil)
				userSvc.EXPECT().CheckUserExists(gomock.Any(), testMembers[0]).Return(true)
				chatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), chatID, testMembers[0]).Return(true, nil)
			},
			expectedError: customerrors.ErrUserAlreadyInChat,
			isErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockChatRepo := mock.NewMockChatRepositoryInterface(ctrl)
			mockUserSvc := mock.NewMockUserInterface(ctrl)
			if tt.mockBehavior != nil {
				tt.mockBehavior(mockChatRepo, mockUserSvc)
			}
			ChatService := service.NewChatService(mockUserSvc, mockChatRepo, nil)
			err := ChatService.AddMembers(context.Background(), tt.chatID, tt.userID, tt.members)
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

func TestRemoveMember(t *testing.T) {
	chatID := int64(1)
	userID := int64(1)
	tests := []struct {
		name          string
		chatID        int64
		userID        int64
		mockBehavior  func(chatRepo *mock.MockChatRepositoryInterface)
		expectedError error
		isErr         bool
	}{
		{
			name:   "Successful remove member",
			chatID: chatID,
			userID: userID,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {
				chatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), chatID, userID).Return(true, nil)
				chatRepo.EXPECT().RemoveMember(gomock.Any(), chatID, userID).Return(nil)
			},
			expectedError: nil,
			isErr:         false,
		},
		{
			name:   "User is not member of chat",
			chatID: chatID,
			userID: userID,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {
				chatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), chatID, userID).Return(false, nil)
			},
			expectedError: customerrors.ErrUserNotMemberOfChat,
			isErr:         true,
		},
		{
			name:   "Invalid input",
			chatID: -1,
			userID: 0,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {

			},
			expectedError: customerrors.ErrInvalidInput,
			isErr:         true,
		},
		{
			name:   "Repository error during membership check",
			chatID: chatID,
			userID: userID,
			mockBehavior: func(chatRepo *mock.MockChatRepositoryInterface) {
				chatRepo.EXPECT().CheckIsMemberOfChat(gomock.Any(), chatID, userID).Return(false, customerrors.ErrFailedToCheck)
			},
			expectedError: customerrors.ErrFailedToCheck,
			isErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockChatRepo := mock.NewMockChatRepositoryInterface(ctrl)
			if tt.mockBehavior != nil {
				tt.mockBehavior(mockChatRepo)
			}
			ChatService := service.NewChatService(nil, mockChatRepo, nil)
			err := ChatService.RemoveMember(context.Background(), tt.chatID, tt.userID)
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
