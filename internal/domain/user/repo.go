package user

import "context"

type Repository interface {
	RegisterUser(ctx context.Context, username string, email string, password string) (User, error)
	SearchUser(ctx context.Context, q string, limit, offset int) ([]User, error)
}
