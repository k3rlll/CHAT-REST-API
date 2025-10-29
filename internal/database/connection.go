package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBConnection struct {
	Conn context.Context
}

func NewDBConnection(ctx context.Context) *DBConnection {
	return &DBConnection{
		Conn: ctx,
	}
}

func (db *DBConnection) ConnectionDB() (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(db.Conn, "postgres://user:password@localhost:5432/chatdb")
	if err != nil {
		return nil, err
	}
	return pool, nil

}
