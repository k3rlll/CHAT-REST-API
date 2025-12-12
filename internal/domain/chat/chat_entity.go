package chat

import "time"

type Chat struct {
	Id               int64     `json:"chat_id" `
	Title            string    `json:"title"`
	IsPrivate        bool      `json:"is_private"`
	CreatedAt        time.Time `json:"created_at"`
	MembersID        []int64   `json:"members"`
	MembersUsernames []string  `json:"members_usernames"`
	MembersCount     int       `json:"members_count"`
}
