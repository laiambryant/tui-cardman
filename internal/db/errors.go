package db

import "fmt"

type ApplyMigrationError struct {
	Path string
	Err  error
}

func (e *ApplyMigrationError) Error() string {
	return fmt.Sprintf("apply %s: %v", e.Path, e.Err)
}

func (e *ApplyMigrationError) Unwrap() error {
	return e.Err
}

type QueryFailedError struct {
	Err error
}

func (e *QueryFailedError) Error() string {
	return fmt.Sprintf("query failed: %v", e.Err)
}

func (e *QueryFailedError) Unwrap() error {
	return e.Err
}

type ExecFailedError struct {
	Err error
}

func (e *ExecFailedError) Error() string {
	return fmt.Sprintf("exec failed: %v", e.Err)
}

func (e *ExecFailedError) Unwrap() error {
	return e.Err
}

type FailedToBeginTransactionError struct {
	Err error
}

func (e *FailedToBeginTransactionError) Error() string {
	return fmt.Sprintf("failed to begin transaction: %v", e.Err)
}

func (e *FailedToBeginTransactionError) Unwrap() error {
	return e.Err
}

type FailedToCommitTransactionError struct {
	Err error
}

func (e *FailedToCommitTransactionError) Error() string {
	return fmt.Sprintf("failed to commit transaction: %v", e.Err)
}

func (e *FailedToCommitTransactionError) Unwrap() error {
	return e.Err
}

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
