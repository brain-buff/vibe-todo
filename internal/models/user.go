package models

import "time"

// User — пользователь приложения.
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserInput — тело запроса на создание пользователя.
type UserInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
