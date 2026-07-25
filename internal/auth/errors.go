package auth

import "errors"

var (
	ErrEmailRequired = errors.New("email is required")
	ErrUserNotFound  = errors.New("user not found")
	ErrUserExists    = errors.New("user with this email already exists")
)
