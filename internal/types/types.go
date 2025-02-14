package types

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username" validate:"required"`
	Email     string `json:"email" validate:"required"`
	Password  string `json:"-"`
	CreatedAt string `json:"created_at"`
}
