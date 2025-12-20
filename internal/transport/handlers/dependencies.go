package handlers

import (
	"context"
	domMess "main/internal/domain/message"
	domUser "main/internal/domain/user"
)

type Manager interface {
	Parse(accessToken string) (int64, error)
	Exists(ctx context.Context, token string) (bool, error)
}

type MessageService interface {
	Send(ctx context.Context, chatID int64, senderID int64, senderUsername string, text string) (domMess.Message, error)
	DeleteMessage(ctx context.Context, messageID int64) error
	Edit(ctx context.Context, messageID int64, newText string) error
	List(ctx context.Context, chatID int64) ([]domMess.Message, error)
}

type UserService interface {
	SearchUser(ctx context.Context, message string) ([]domUser.User, error)
	RegisterUser(ctx context.Context, username, email, password string) (domUser.User, error)
}

type AuthService interface {
	LoginUser(ctx context.Context, userID int64, password string) (string, string, error)
	LogoutUser(ctx context.Context, userID int64, refreshToken string) error
}
type ChatService interface {
	CreateChat(ctx context.Context, isPrivate bool, title string, members []int64) (domUser.User, error)
	ListOfChats(ctx context.Context, userId int64) ([]domUser.User, error)
	OpenChat(ctx context.Context, chatID int64, userId int64) (domUser.User, error)
	GetChatDetails(ctx context.Context, chatID int64) (domUser.User, error)
	DeleteChat(ctx context.Context, chatID int64) error
	AddMembers(ctx context.Context, chatID int64, userID int64, members []int64) error
}

type JWTManager interface {
	Parse(accessToken string) (int64, error)
	Exists(ctx context.Context, token string) (bool, error)
}
