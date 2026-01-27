package cardgame

import "fmt"

type FailedToQueryCardGamesError struct {
	Err error
}

func (e *FailedToQueryCardGamesError) Error() string {
	return fmt.Sprintf("failed to query card games: %v", e.Err)
}

func (e *FailedToQueryCardGamesError) Unwrap() error {
	return e.Err
}

type FailedToScanCardGameError struct {
	Err error
}

func (e *FailedToScanCardGameError) Error() string {
	return fmt.Sprintf("failed to scan card game: %v", e.Err)
}

func (e *FailedToScanCardGameError) Unwrap() error {
	return e.Err
}

type ErrorIteratingCardGamesError struct {
	Err error
}

func (e *ErrorIteratingCardGamesError) Error() string {
	return fmt.Sprintf("error iterating card games: %v", e.Err)
}

func (e *ErrorIteratingCardGamesError) Unwrap() error {
	return e.Err
}
