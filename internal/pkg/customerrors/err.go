package customerrors

import "errors"

var (
	ErrDecodingRequestBody   = errors.New("failed to decode request body")
	ErrExpiredToken          = errors.New("expired token")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrMessageDoesNotExists  = errors.New("the messsage does not exists")
	ErrUserNotMemberOfChat   = errors.New("user is not a member of the chat")
	ErrFailedToCheck         = errors.New("failed to check")
	ErrUserNotFound          = errors.New("user not found")
	ErrUserAlreadyInChat     = errors.New("user is already in the chat")
	ErrSecretKeyNotSet       = errors.New("secret key is not set")
	ErrFailedToSaveToken     = errors.New("failed to save token")
	ErrTokenCreationFailed   = errors.New("token creation failed")
	ErrRedisFailed           = errors.New("redis failed")
	ErrInvalidInput          = errors.New("invalid input")
)
