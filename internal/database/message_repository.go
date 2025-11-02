package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Message struct{
	id int
	text string
	createdAt time.Duration
	chatID int 
	userID int
}

type messageRepository struct {
	pool *pgxpool.Pool
	ctx  context.Context
}

func NewMessageRepository(pool *pgxpool.Pool, ctx context.Context) *messageRepository {
	return &messageRepository{
		pool: pool,
		ctx:  ctx,
	}
}

type MessageRepository interface{

}


func (m *messageRepository) DeleteMessage(id int) error {

	return nil
}