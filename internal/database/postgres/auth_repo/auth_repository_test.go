package auth_repo

import (
	dbtest "main/internal/database/postgres/repository_test"
	"testing"
)

func TestAuthRepo_Login(t *testing.T) {
	pool, teardown := dbtest.SetupTestDB(t)
	defer teardown()
	repo := NewAuthRepository(pool, nil)

	// Add test user
	user := dbtest.TestUser{
		Username:     "testuser",
		Email:        ""
}
