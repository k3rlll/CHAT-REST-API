package customerrors

import "errors"

var (
	ErrDecodingRequestBody       = errors.New("failed to decode request body")
	ErrInvalidNicknameOrPassword = errors.New("invalid nickname or password")
	ErrExpiredToken              = errors.New("expired token")
	ErrEmailAlreadyExists        = errors.New("email already exists")
	ErrInvalidPassword           = errors.New("password does not meet complexity requirements")
	ErrMessageDoesNotExists      = errors.New("the messsage does not exists")
	ErrEmptyQuery                = errors.New("search query is empty")
	ErrUserNotMemberOfChat       = errors.New("user is not a member of the chat")
	ErrMessageIsEmpty            = errors.New("message text is empty")
)
