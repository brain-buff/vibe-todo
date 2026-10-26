package models

import "github.com/golang-jwt/jwt/v5"

// Claims — JWT claims с ID пользователя.
type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthResponse — ответ на регистрацию/логин.
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
