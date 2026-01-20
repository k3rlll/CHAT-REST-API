package mock_test

import (
	"errors"
	"io"
	"log/slog"
	dom "main/internal/domain/entity"
	service "main/internal/usecase/user"
	. "main/internal/usecase/user/mock"
	"main/pkg/customerrors"
	"testing"

	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func TestRegisterUser(t *testing.T) {

	testUsername := "testuser"
	testEmail := "test@test.com"
	validPassword := "StrongP@ssw0rd!"

	tests := []struct {
		name          string
		mockBehavior  func(m *MockUserInterface)
		inputUsername string
		inputEmail    string
		inputPassword string
		expectedUser  dom.User
		expectedError error
		isErr         bool
	}{
		{
			name:          "Successful registration",
			inputUsername: testUsername,
			inputEmail:    testEmail,
			inputPassword: validPassword,
			mockBehavior: func(m *MockUserInterface) {
				m.EXPECT().RegisterUser(gomock.Any(), testUsername, testEmail, gomock.Any()).
					Return(dom.User{
						Username: testUsername,
						Email:    testEmail,
					}, nil)
			},
			expectedUser: dom.User{
				Username: testUsername,
				Email:    testEmail,
			},
			isErr: false,
		},
		{
			name:          "Database error on registration",
			inputUsername: testUsername,
			inputEmail:    testEmail,
			inputPassword: validPassword,
			mockBehavior: func(m *MockUserInterface) {
				m.EXPECT().RegisterUser(gomock.Any(), testUsername, testEmail, gomock.Any()).
					Return(dom.User{}, errors.New("database error"))
			},
			expectedUser:  dom.User{},
			expectedError: customerrors.ErrDatabase,
			isErr:         true,
		},
		{
			name:          "Invalid password",
			inputUsername: testUsername,
			inputEmail:    testEmail,
			inputPassword: "123",
			mockBehavior: func(m *MockUserInterface) {

			},
			expectedUser:  dom.User{},
			expectedError: customerrors.ErrInvalidInput,
			isErr:         true,
		},
	}

	sillentLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockUserInterface := NewMockUserInterface(ctrl)

			if tt.mockBehavior != nil {
				tt.mockBehavior(mockUserInterface)
			}

			userService := service.NewUserService(mockUserInterface, sillentLogger)

			user, err := userService.RegisterUser(nil, tt.inputUsername, tt.inputEmail, tt.inputPassword)

			if tt.isErr {

				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				} else {
					assert.Error(t, err)
				}
			} else {

				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUser, user)
			}
		})
	}
}
