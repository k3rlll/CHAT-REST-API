package chat

import "time"

type Chat struct {
	Id           int64     `json:"chat_id" `
	Title        string    `json:"title"`
	IsPrivate    bool      `json:"is_private"`
	CreatedAt    time.Time `json:"created_at"`
	Members      []string  `json:"members"`
	MembersCount int       `json:"members_count"`
}
