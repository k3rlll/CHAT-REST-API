package chat

import "time"

type Chat struct {
	Id        int
	Title     string
	IsPrivate bool
	CreatedAt time.Time
}
