package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second),
		))
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "testdb", "testuser", "password")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create connection pool: %v", err)
	}

	initSchema(t, pool)

	teardown := func() {
		pool.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %v", err)
		}
	}
	return pool, teardown
}

func initSchema(t *testing.T, pool *pgxpool.Pool) {
	schema := `
	CREATE TABLE users(
		id BIGSERIAL PRIMARY KEY,
		username VARCHAR(20) NOT NULL UNIQUE,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash TEXT NOT NULL
	);

	CREATE TABLE chats (
		id BIGSERIAL PRIMARY KEY,
		title VARCHAR(20),
		is_private BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);


	CREATE TABLE chat_members (
		id BIGSERIAL PRIMARY KEY,
		chat_id BIGINT NOT NULL,
		user_id BIGINT NOT NULL,
		joined_at TIMESTAMPTZ DEFAULT NOW(),
		UNIQUE(chat_id, user_id),
		FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);


	CREATE TABLE messages(
		id BIGSERIAL PRIMARY KEY,
		text TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		chat_id BIGINT NOT NULL,
		sender_id BIGINT NOT NULL,
		FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
		FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
	);


	CREATE TABLE refresh_tokens (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		token TEXT NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	_, err := pool.Exec(context.Background(), schema)
	if err != nil {
		t.Fatalf("failed to initialize schema: %v", err)
	}
}
