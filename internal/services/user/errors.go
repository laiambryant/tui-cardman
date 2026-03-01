package user

import "errors"

var ErrUserNotFound = errors.New("user not found")
var ErrNoUsersFound = errors.New("no users found")
