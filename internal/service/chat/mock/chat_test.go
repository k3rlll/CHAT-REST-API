package mock

import (
	context "context"
	"io"
	"log/slog"
	dom "main/internal/domain/entity"
	"main/internal/pkg/customerrors"
	service "main/internal/service/chat"
	"testing"
	"time"

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
		mockBehavior  func(chatRepo *MockChatRepositoryInterface)
		isErr         bool
		expectedChat  dom.Chat
		expectedError error
	}{
		{
			name:      "Successful chat creation",
			title:     title,
			isPrivate: false,
			members:   members,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
				chatRepo.EXPECT().CreateChat(gomock.Any(), title, false, members).Return(int64(1), nil)
			},
			expectedChat: dom.Chat{
				Id:        1,
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
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {

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
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {

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
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
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
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
			},
			expectedChat:  dom.Chat{},
			expectedError: customerrors.ErrInvalidInput,
			isErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockChatRepo := NewMockChatRepositoryInterface(ctrl)

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
		mockBehavior  func(chatRepo *MockChatRepositoryInterface)
		expectedError error
		isErr         bool
	}{
		{
			name:   "Successful chat deletion",
			chatID: chatID,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
				chatRepo.EXPECT().CheckIfChatExists(gomock.Any(), chatID).Return(true, nil)
				chatRepo.EXPECT().DeleteChat(gomock.Any(), chatID).Return(nil)
			},
			expectedError: nil,
			isErr:         false,
		},
		{
			name:   "Chat does not exist",
			chatID: chatID,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
				chatRepo.EXPECT().CheckIfChatExists(gomock.Any(), chatID).Return(false, nil)
			},
			expectedError: customerrors.ErrNotFound,
			isErr:         true,
		},
		{
			name:   "Repository error during existence check",
			chatID: chatID,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
				chatRepo.EXPECT().CheckIfChatExists(gomock.Any(), chatID).Return(false, customerrors.ErrDatabase)
			},
			expectedError: customerrors.ErrDatabase,
			isErr:         true,
		},
		{
			name:   "Repository error during chat deletion",
			chatID: chatID,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
				chatRepo.EXPECT().CheckIfChatExists(gomock.Any(), chatID).Return(true, nil)
				chatRepo.EXPECT().DeleteChat(gomock.Any(), chatID).Return(customerrors.ErrDatabase)
			},
			expectedError: customerrors.ErrDatabase,
			isErr:         true,
		},
		{
			name:   "Invalid chat ID",
			chatID: -1,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
			},
			expectedError: customerrors.ErrInvalidInput,
			isErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockChatRepo := NewMockChatRepositoryInterface(ctrl)
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
	sillentLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	chatID := int64(1)
	userID := int64(1)
	testtime := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	testChat := dom.Chat{
		Id:               chatID,
		Title:            "Test Chat",
		IsPrivate:        false,
		CreatedAt:        testtime,
		MembersID:        []int64{1, 2, 3},
		MembersUsernames: []string{"user1", "user2", "user3"},
		MembersCount:     3,
	}
	testMessages := []dom.Message{
		{Id: 1, ChatID: chatID, SenderID: 2, SenderUsername: "user2", Text: "Hello", CreatedAt: testtime},
		{Id: 2, ChatID: chatID, SenderID: 3, SenderUsername: "user3", Text: "Hi", CreatedAt: testtime},
	}

	tests := []struct {
		name             string
		chatID           int64
		userID           int64
		mockBehavior     func(chatRepo *MockChatRepositoryInterface)
		isMemberExpect   bool
		expectedChat     dom.Chat
		expectedMessages []dom.Message
		expectedError    error
		isErr            bool
	}{
		{
			name:   "Successful OpenChat",
			chatID: chatID,
			userID: userID,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, userID).Return(true, nil)
				chatRepo.EXPECT().GetChatDetails(gomock.Any(), chatID).Return(dom.Chat{
					Id:               chatID,
					Title:            testChat.Title,
					IsPrivate:        testChat.IsPrivate,
					CreatedAt:        testChat.CreatedAt,
					MembersID:        []int64{1, 2, 3},
					MembersUsernames: []string{"user1", "user2", "user3"},
					MembersCount:     testChat.MembersCount,
				}, nil)
				chatRepo.EXPECT().OpenChat(gomock.Any(), chatID, userID).Return(testMessages, nil)
			},
			expectedChat:     testChat,
			expectedMessages: testMessages,
			expectedError:    nil,
			isErr:            false,
		},
		{
			name:   "Repository error during get chat details",
			chatID: chatID,
			userID: userID,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, userID).Return(true, nil)
				chatRepo.EXPECT().GetChatDetails(gomock.Any(), chatID).Return(dom.Chat{}, customerrors.ErrDatabase)
			},
			expectedChat:  dom.Chat{},
			expectedError: customerrors.ErrDatabase,
			isErr:         true,
		},
		{
			name:   "Invalid chat ID",
			chatID: -1,
			userID: userID,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
			},
			expectedChat:     dom.Chat{},
			expectedMessages: []dom.Message{},
			expectedError:    customerrors.ErrInvalidInput,
			isErr:            true,
		},
		{
			name:   "User not a member of the chat",
			chatID: chatID,
			userID: userID,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, userID).Return(false, nil)
			},
			isMemberExpect:   false,
			expectedChat:     dom.Chat{},
			expectedMessages: []dom.Message{},
			expectedError:    customerrors.ErrUserNotMemberOfChat,
			isErr:            true,
		},
		{
			name:   "Error checking if user is member of chat",
			chatID: chatID,
			userID: userID,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, userID).Return(false, customerrors.ErrFailedToCheck)
			},
			isMemberExpect:   false,
			expectedChat:     dom.Chat{},
			expectedMessages: []dom.Message{},
			expectedError:    customerrors.ErrFailedToCheck,
			isErr:            true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockChatRepo := NewMockChatRepositoryInterface(ctrl)
			if tt.mockBehavior != nil {
				tt.mockBehavior(mockChatRepo)
			}
			ChatService := service.NewChatService(nil, mockChatRepo, sillentLogger)
			chat, message, err := ChatService.OpenChat(context.Background(), tt.chatID, tt.userID)

			if tt.isErr {
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMessages, message)
				assert.Equal(t, tt.expectedChat, chat)
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
		mockBehavior  func(chatRepo *MockChatRepositoryInterface, userSvc *MockUserInterface)
		expectedError error
		isErr         bool
	}{
		{
			name:    "Successful add members",
			chatID:  chatID,
			userID:  userID,
			members: testMembers,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface, userSvc *MockUserInterface) {
				//проверка инициатора на членство в чате
				userSvc.EXPECT().CheckUserExists(gomock.Any(), userID).Return(true)
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, userID).Return(true, nil)
				//проверка каждого нового участника на существование и членство в чате
				userSvc.EXPECT().CheckUserExists(gomock.Any(), testMembers[0]).Return(true)
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, testMembers[0]).Return(false, nil)

				userSvc.EXPECT().CheckUserExists(gomock.Any(), testMembers[1]).Return(true)
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, testMembers[1]).Return(false, nil)

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
			mockBehavior: func(chatRepo *MockChatRepositoryInterface, userSvc *MockUserInterface) {
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
			mockBehavior: func(chatRepo *MockChatRepositoryInterface, userSvc *MockUserInterface) {
				userSvc.EXPECT().CheckUserExists(gomock.Any(), userID).Return(true)
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, userID).Return(false, nil)
			},
			expectedError: customerrors.ErrUserNotMemberOfChat,
			isErr:         true,
		},
		{
			name:    "New member not found",
			chatID:  chatID,
			userID:  userID,
			members: testMembers,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface, userSvc *MockUserInterface) {
				userSvc.EXPECT().CheckUserExists(gomock.Any(), userID).Return(true)
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, userID).Return(true, nil)
				//проверка каждого нового участника на существование и членство в чате
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
			mockBehavior: func(chatRepo *MockChatRepositoryInterface, userSvc *MockUserInterface) {
				userSvc.EXPECT().CheckUserExists(gomock.Any(), userID).Return(true)
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, userID).Return(true, nil)
				//проверка каждого нового участника на существование и членство в чате
				userSvc.EXPECT().CheckUserExists(gomock.Any(), testMembers[0]).Return(true)
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, testMembers[0]).Return(true, nil)
			},
			expectedError: customerrors.ErrUserAlreadyInChat,
			isErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockChatRepo := NewMockChatRepositoryInterface(ctrl)
			mockUserSvc := NewMockUserInterface(ctrl)
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
		mockBehavior  func(chatRepo *MockChatRepositoryInterface)
		expectedError error
		isErr         bool
	}{
		{
			name:   "Successful remove member",
			chatID: chatID,
			userID: userID,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, userID).Return(true, nil)
				chatRepo.EXPECT().RemoveMember(gomock.Any(), chatID, userID).Return(nil)
			},
			expectedError: nil,
			isErr:         false,
		},
		{
			name:   "User is not member of chat",
			chatID: chatID,
			userID: userID,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, userID).Return(false, nil)
			},
			expectedError: customerrors.ErrUserNotMemberOfChat,
			isErr:         true,
		},
		{
			name:   "Invalid input",
			chatID: -1,
			userID: 0,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {

			},
			expectedError: customerrors.ErrInvalidInput,
			isErr:         true,
		},
		{
			name:   "Repository error during membership check",
			chatID: chatID,
			userID: userID,
			mockBehavior: func(chatRepo *MockChatRepositoryInterface) {
				chatRepo.EXPECT().UserInChat(gomock.Any(), chatID, userID).Return(false, customerrors.ErrFailedToCheck)
			},
			expectedError: customerrors.ErrFailedToCheck,
			isErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockChatRepo := NewMockChatRepositoryInterface(ctrl)
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
