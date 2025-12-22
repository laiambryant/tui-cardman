package auth

import "time"

type User struct {
	ID           int64
	Name         string
	Surname      string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLogin    *time.Time
	Active       bool
}

type LoginRequest struct {
	Email    string
	Password string
}

type RegisterRequest struct {
	Name     string
	Surname  string
	Email    string
	Password string
}
