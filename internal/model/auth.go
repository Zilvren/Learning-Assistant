package model

import "time"

type User struct {
	ID        int64      `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email,omitempty"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
	LastLogin *time.Time `json:"-"`
}

type AuthUser struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	Status       string
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type AuthResponse struct {
	User User `json:"user"`
}
