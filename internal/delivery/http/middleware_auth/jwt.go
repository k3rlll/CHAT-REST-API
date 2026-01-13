package middleware_auth

import (
	"context"
	"fmt"
	"main/internal/pkg/customerrors"
)

func GetUserIDFromContext(ctx context.Context) (int64, error) {
	userID, ok := ctx.Value("user_id").(int64)
	if !ok {
		return 0, fmt.Errorf("user id not found in context: %w", customerrors.ErrUserNotFound)
	}
	return userID, nil
}
