package tui

import "fmt"

type FailedToInitializeConfigManagerError struct {
	Err error
}

func (e *FailedToInitializeConfigManagerError) Error() string {
	return fmt.Sprintf("failed to initialize config manager: %v", e.Err)
}

func (e *FailedToInitializeConfigManagerError) Unwrap() error {
	return e.Err
}

type FailedToLoadCardGamesError struct {
	Err error
}

func (e *FailedToLoadCardGamesError) Error() string {
	return fmt.Sprintf("failed to load card games: %v", e.Err)
}

func (e *FailedToLoadCardGamesError) Unwrap() error {
	return e.Err
}

type FailedToCheckForExistingUsersError struct {
	Err error
}

func (e *FailedToCheckForExistingUsersError) Error() string {
	return fmt.Sprintf("failed to check for existing users: %v", e.Err)
}

func (e *FailedToCheckForExistingUsersError) Unwrap() error {
	return e.Err
}

type FailedToGetFirstUserForLocalModeError struct {
	Err error
}

func (e *FailedToGetFirstUserForLocalModeError) Error() string {
	return fmt.Sprintf("failed to get first user for local mode: %v", e.Err)
}

func (e *FailedToGetFirstUserForLocalModeError) Unwrap() error {
	return e.Err
}

type FailedToLoadCardsError struct {
	Err error
}

func (e *FailedToLoadCardsError) Error() string {
	return fmt.Sprintf("failed to load cards: %v", e.Err)
}

func (e *FailedToLoadCardsError) Unwrap() error {
	return e.Err
}

type FailedToLoadUserCollectionError struct {
	Err error
}

func (e *FailedToLoadUserCollectionError) Error() string {
	return fmt.Sprintf("failed to load user collection: %v", e.Err)
}

func (e *FailedToLoadUserCollectionError) Unwrap() error {
	return e.Err
}

type FailedToSaveConfigurationError struct {
	Err error
}

func (e *FailedToSaveConfigurationError) Error() string {
	return fmt.Sprintf("failed to save configuration: %v", e.Err)
}

func (e *FailedToSaveConfigurationError) Unwrap() error {
	return e.Err
}
