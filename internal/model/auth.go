package model

import "time"

type User struct {
	ID            int64      `json:"id"`
	Username      string     `json:"username"`
	Email         string     `json:"email,omitempty"`
	EmailVerified bool       `json:"email_verified"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"-"`
	UpdatedAt     time.Time  `json:"-"`
	LastLogin     *time.Time `json:"-"`
}

type AuthUser struct {
	ID            int64
	Username      string
	Email         string
	EmailVerified bool
	PasswordHash  string
	Status        string
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

type VerifyEmailRequest struct {
	Token string `json:"token"`
}

type ResendVerificationRequest struct {
	Email string `json:"email"`
}

type AuthResponse struct {
	User User `json:"user"`
}

type RegistrationResponse struct {
	EmailVerificationRequired bool   `json:"email_verification_required"`
	Email                     string `json:"email"`
}
