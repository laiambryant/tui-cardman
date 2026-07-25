package user

import (
	"errors"

	"github.com/laiambryant/tui-cardman/internal/auth"
)

var (
	ErrUserNotFound = auth.ErrUserNotFound
	ErrNoUsersFound = errors.New("no users found")
)
