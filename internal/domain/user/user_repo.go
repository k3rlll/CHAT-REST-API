package user

import "context"

type UserRepository interface {
	RegisterUser(ctx context.Context, username string, email string, password string) (User, error)
	SearchUser(ctx context.Context, q string) ([]User, error)
	CheckUsernameExists(ctx context.Context, username string) bool
	ChangeUsername(ctx context.Context, username string) (User, error)
	CheckUserExists(ctx context.Context, userID int64) bool
}
