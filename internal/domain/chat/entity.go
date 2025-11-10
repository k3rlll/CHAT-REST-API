package chat

import "time"

type Chat struct {
	Id        int
	Name      string
	IsPrivate bool
	CreatedAt time.Time
}
