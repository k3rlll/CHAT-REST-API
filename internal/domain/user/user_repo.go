package user

import "context"

type UserRepository interface {
	RegisterUser(ctx context.Context, username string, nickname string, email string, password string) (User, error)
	SearchUser(ctx context.Context, q string) ([]User, error)
	CheckUsernameExists(ctx context.Context, username string) bool
}
