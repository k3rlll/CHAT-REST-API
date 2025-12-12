package handlers


import "context"

type Manager interface {
	Parse(accessToken string) (int64, error)
	Exists(ctx context.Context, token string) (bool, error)
}