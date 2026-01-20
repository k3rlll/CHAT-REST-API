package mock_test

import (
	context "context"
	"errors"
	"io"
	"log/slog"
	dom "main/internal/domain/entity"
	events "main/internal/domain/events"
	service "main/internal/usecase/message"
	mock "main/internal/usecase/message/mock"
	"main/pkg/customerrors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSendMessage(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChat := mock.NewMockChatInterface(ctrl)
	mockMsgRepo := mock.NewMockMessageRepository(ctrl)
	mockKafka := mock.NewMockKafkaProducer(ctrl)

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	tests := []struct {
		name           string
		chatID         int64
		userID         int64
		senderUsername string
		text           string
		setup          func()
		wantErr        error
	}{
		{
			name:           "Successful message sending",
			chatID:         1,
			userID:         10,
			senderUsername: "senior_dev",
			text:           "Hello, world!",
			setup: func() {

				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), int64(1), int64(10)).
					Return(true, nil)
				mockMsgRepo.EXPECT().
					SaveMessage(gomock.Any(), gomock.AssignableToTypeOf(dom.Message{})).
					Return("mongo_id_123", nil)
				mockKafka.EXPECT().
					SendMessageCreated(gomock.Any(), gomock.AssignableToTypeOf(events.MessageCreated{})).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name:   "Error: user is not a member of the chat",
			chatID: 1,
			userID: 99,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), int64(1), int64(99)).
					Return(false, nil)
			},
			wantErr: customerrors.ErrUserNotMemberOfChat,
		},
		{
			name:   "Error: database failed to save message",
			chatID: 1,
			userID: 10,
			setup: func() {
				mockChat.EXPECT().CheckIsMemberOfChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				mockMsgRepo.EXPECT().SaveMessage(gomock.Any(), gomock.Any()).Return("", errors.New("mongo down"))
			},
			wantErr: customerrors.ErrDatabase,
		},
		{
			name:   "Error: database failure when checking membership",
			chatID: 1,
			userID: 10,
			setup: func() {
				mockChat.EXPECT().CheckIsMemberOfChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				mockMsgRepo.EXPECT().SaveMessage(gomock.Any(), gomock.Any()).Return("id123", nil)
				mockKafka.EXPECT().SendMessageCreated(gomock.Any(), gomock.Any()).Return(errors.New("kafka connection error"))
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			service := &service.MessageService{
				Chat:   mockChat,
				Msg:    mockMsgRepo,
				Kafka:  mockKafka,
				Logger: logger,
			}

			msg, err := service.SendMessage(context.Background(), tt.chatID, tt.userID, tt.senderUsername, tt.text)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, msg)
			}
		})
	}
}

func TestDeleteMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	mockChat := mock.NewMockChatInterface(ctrl)
	mockMsgRepo := mock.NewMockMessageRepository(ctrl)

	validIDs := []string{"651eb123", "651eb456"}
	var senderID int64 = 10
	var chatID int64 = 1

	tests := []struct {
		name     string
		senderID int64
		chatID   int64
		msgIDs   []string
		setup    func()
		wantErr  error
	}{
		{
			name:     "Successful message deletion",
			senderID: senderID,
			chatID:   chatID,
			msgIDs:   validIDs,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), chatID, senderID).
					Return(true, nil)

				mockMsgRepo.EXPECT().
					DeleteMessage(gomock.Any(), senderID, chatID, validIDs).
					Return(int64(2), nil)
			},
			wantErr: nil,
		},
		{
			name:     "Error: invalid input (empty ID list)",
			senderID: senderID,
			chatID:   chatID,
			msgIDs:   []string{},
			setup:    func() {},
			wantErr:  customerrors.ErrInvalidInput,
		},
		{
			name:     "Error: user is not a member of the chat",
			senderID: 666,
			chatID:   chatID,
			msgIDs:   validIDs,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), chatID, int64(666)).
					Return(false, nil)
			},
			wantErr: customerrors.ErrUserNotMemberOfChat,
		},
		{
			name:     "Error: messages not found (deletedCount == 0)",
			senderID: senderID,
			chatID:   chatID,
			msgIDs:   validIDs,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), chatID, senderID).
					Return(true, nil)

				mockMsgRepo.EXPECT().
					DeleteMessage(gomock.Any(), senderID, chatID, validIDs).
					Return(int64(0), nil)
			},
			wantErr: customerrors.ErrMessageDoesNotExists,
		},
		{
			name:     "Error: repository failure during deletion",
			senderID: senderID,
			chatID:   chatID,
			msgIDs:   validIDs,
			setup: func() {
				mockChat.EXPECT().CheckIsMemberOfChat(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				mockMsgRepo.EXPECT().
					DeleteMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("mongo connection lost"))
			},
			wantErr: errors.New("mongo connection lost"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			service := &service.MessageService{
				Chat:   mockChat,
				Msg:    mockMsgRepo,
				Logger: logger,
			}

			err := service.DeleteMessage(context.Background(), tt.senderID, tt.chatID, tt.msgIDs)

			if tt.wantErr != nil {
				assert.Error(t, err)
				// Проверяем, содержит ли обернутая ошибка наш ожидаемый тип
				assert.Contains(t, err.Error(), tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEditMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChat := mock.NewMockChatInterface(ctrl)
	mockMsgRepo := mock.NewMockMessageRepository(ctrl)

	msgID := "651eb1234567890abcdef123"
	newText := "Updated text"

	tests := []struct {
		name     string
		senderID int64
		chatID   int64
		msgID    string
		newText  string
		setup    func()
		wantErr  error
	}{
		{
			name:     "Success",
			senderID: 10,
			chatID:   1,
			msgID:    msgID,
			newText:  newText,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), int64(1), int64(10)).
					Return(true, nil)
				mockMsgRepo.EXPECT().
					EditMessage(gomock.Any(), int64(10), int64(1), msgID, newText).
					Return(int64(1), nil)
			},
			wantErr: nil,
		},
		{
			name:     "Invalid input",
			senderID: 0,
			chatID:   1,
			msgID:    "",
			newText:  "",
			setup:    func() {},
			wantErr:  customerrors.ErrInvalidInput,
		},
		{
			name:     "Membership check database error",
			senderID: 10,
			chatID:   1,
			msgID:    msgID,
			newText:  newText,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(false, errors.New("sql error"))
			},
			wantErr: customerrors.ErrDatabase,
		},
		{
			name:     "User is not a member",
			senderID: 999,
			chatID:   1,
			msgID:    msgID,
			newText:  newText,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), int64(1), int64(999)).
					Return(false, nil)
			},
			wantErr: customerrors.ErrUserNotMemberOfChat,
		},
		{
			name:     "Message not found or not an author",
			senderID: 10,
			chatID:   1,
			msgID:    msgID,
			newText:  newText,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), int64(1), int64(10)).
					Return(true, nil)
				mockMsgRepo.EXPECT().
					EditMessage(gomock.Any(), int64(10), int64(1), msgID, newText).
					Return(int64(0), nil)
			},
			wantErr: customerrors.ErrMessageDoesNotExists,
		},
		{
			name:     "Edit repository error",
			senderID: 10,
			chatID:   1,
			msgID:    msgID,
			newText:  newText,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), int64(1), int64(10)).
					Return(true, nil)
				mockMsgRepo.EXPECT().
					EditMessage(gomock.Any(), int64(10), int64(1), msgID, newText).
					Return(int64(0), errors.New("mongo timeout"))
			},
			wantErr: errors.New("mongo timeout"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			service := &service.MessageService{
				Chat:   mockChat,
				Msg:    mockMsgRepo,
				Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
			}

			err := service.EditMessage(context.Background(), tt.senderID, tt.chatID, tt.msgID, tt.newText)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChat := mock.NewMockChatInterface(ctrl)
	mockMsgRepo := mock.NewMockMessageRepository(ctrl)

	now := "2024-06-15T12:00:00Z"
	anchorID := "651eb1234567890abcdef123"
	limit := int64(20)
	messages := []dom.Message{{Text: "old message"}, {Text: "new message"}}

	tests := []struct {
		name     string
		userID   int64
		chatID   int64
		limit    int64
		setup    func()
		wantMsgs []dom.Message
		wantErr  error
	}{
		{
			name:   "Success",
			userID: 10,
			chatID: 1,
			limit:  limit,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), int64(1), int64(10)).
					Return(true, nil)
				mockMsgRepo.EXPECT().
					GetMessages(gomock.Any(), int64(1), now, anchorID, limit).
					Return(messages, nil)
			},
			wantMsgs: messages,
			wantErr:  nil,
		},
		{
			name:     "Invalid input",
			userID:   0,
			chatID:   1,
			limit:    0,
			setup:    func() {},
			wantMsgs: nil,
			wantErr:  customerrors.ErrInvalidInput,
		},
		{
			name:   "Membership database error",
			userID: 10,
			chatID: 1,
			limit:  limit,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(false, errors.New("db connection lost"))
			},
			wantMsgs: nil,
			wantErr:  customerrors.ErrDatabase,
		},
		{
			name:   "User not a member",
			userID: 999,
			chatID: 1,
			limit:  limit,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), int64(1), int64(999)).
					Return(false, nil)
			},
			wantMsgs: nil,
			wantErr:  customerrors.ErrUserNotMemberOfChat,
		},
		{
			name:   "Repository error",
			userID: 10,
			chatID: 1,
			limit:  limit,
			setup: func() {
				mockChat.EXPECT().
					CheckIsMemberOfChat(gomock.Any(), int64(1), int64(10)).
					Return(true, nil)
				mockMsgRepo.EXPECT().
					GetMessages(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("mongo fetch error"))
			},
			wantMsgs: nil,
			wantErr:  errors.New("mongo fetch error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			service := &service.MessageService{
				Chat: mockChat,
				Msg:  mockMsgRepo,
			}

			got, err := service.GetMessages(context.Background(), tt.userID, tt.chatID, now, anchorID, tt.limit)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMsgs, got)
			}
		})
	}
}
