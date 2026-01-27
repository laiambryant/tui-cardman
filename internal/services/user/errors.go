package user

import "fmt"

type FailedToCreateUserError struct {
	Err error
}

func (e *FailedToCreateUserError) Error() string {
	return fmt.Sprintf("failed to create user: %v", e.Err)
}

func (e *FailedToCreateUserError) Unwrap() error {
	return e.Err
}

type FailedToGetUserIDError struct {
	Err error
}

func (e *FailedToGetUserIDError) Error() string {
	return fmt.Sprintf("failed to get user id: %v", e.Err)
}

func (e *FailedToGetUserIDError) Unwrap() error {
	return e.Err
}

type UserNotFoundError struct{}

func (e *UserNotFoundError) Error() string {
	return "user not found"
}

type FailedToGetUserError struct {
	Err error
}

func (e *FailedToGetUserError) Error() string {
	return fmt.Sprintf("failed to get user: %v", e.Err)
}

func (e *FailedToGetUserError) Unwrap() error {
	return e.Err
}

type FailedToUpdateLastLoginError struct {
	Err error
}

func (e *FailedToUpdateLastLoginError) Error() string {
	return fmt.Sprintf("failed to update last login: %v", e.Err)
}

func (e *FailedToUpdateLastLoginError) Unwrap() error {
	return e.Err
}

type FailedToCountUsersError struct {
	Err error
}

func (e *FailedToCountUsersError) Error() string {
	return fmt.Sprintf("failed to count users: %v", e.Err)
}

func (e *FailedToCountUsersError) Unwrap() error {
	return e.Err
}

type NoUsersFoundError struct{}

func (e *NoUsersFoundError) Error() string {
	return "no users found"
}

type FailedToGetFirstUserError struct {
	Err error
}

func (e *FailedToGetFirstUserError) Error() string {
	return fmt.Sprintf("failed to get first user: %v", e.Err)
}

func (e *FailedToGetFirstUserError) Unwrap() error {
	return e.Err
}
