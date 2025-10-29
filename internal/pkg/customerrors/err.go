package customerrors

import "errors"

var (
	ErrDecodingRequestBody       = errors.New("failed to decode request body")
	ErrInvalidNicknameOrPassword = errors.New("invalid nickname or password")
)
