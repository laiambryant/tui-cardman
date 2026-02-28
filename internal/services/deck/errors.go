package deck

import "fmt"

type FailedToCreateDeckError struct{ Err error }

func (e *FailedToCreateDeckError) Error() string {
	return fmt.Sprintf("failed to create deck: %v", e.Err)
}
func (e *FailedToCreateDeckError) Unwrap() error { return e.Err }

type FailedToQueryDecksError struct{ Err error }

func (e *FailedToQueryDecksError) Error() string {
	return fmt.Sprintf("failed to query decks: %v", e.Err)
}
func (e *FailedToQueryDecksError) Unwrap() error { return e.Err }

type FailedToScanDeckError struct{ Err error }

func (e *FailedToScanDeckError) Error() string { return fmt.Sprintf("failed to scan deck: %v", e.Err) }
func (e *FailedToScanDeckError) Unwrap() error { return e.Err }

type FailedToUpdateDeckError struct{ Err error }

func (e *FailedToUpdateDeckError) Error() string {
	return fmt.Sprintf("failed to update deck: %v", e.Err)
}
func (e *FailedToUpdateDeckError) Unwrap() error { return e.Err }

type FailedToDeleteDeckError struct{ Err error }

func (e *FailedToDeleteDeckError) Error() string {
	return fmt.Sprintf("failed to delete deck: %v", e.Err)
}
func (e *FailedToDeleteDeckError) Unwrap() error { return e.Err }

type FailedToGetDeckQuantitiesError struct{ Err error }

func (e *FailedToGetDeckQuantitiesError) Error() string {
	return fmt.Sprintf("failed to get deck quantities: %v", e.Err)
}
func (e *FailedToGetDeckQuantitiesError) Unwrap() error { return e.Err }

type FailedToUpsertDeckCardError struct{ Err error }

func (e *FailedToUpsertDeckCardError) Error() string {
	return fmt.Sprintf("failed to upsert deck card: %v", e.Err)
}
func (e *FailedToUpsertDeckCardError) Unwrap() error { return e.Err }
