package chat

import "time"

type Chat struct {
	Id           int64
	Title        string
	IsPrivate    bool
	CreatedAt    time.Time
	Members      []string
	MembersCount int
}
