package user

type User struct {
	ID       int64  `json:"id"`
	Nickname string `json:"nickname"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
