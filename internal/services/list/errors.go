package list

import "fmt"

type FailedToCreateListError struct {
	Err error
}

func (e *FailedToCreateListError) Error() string {
	return fmt.Sprintf("failed to create list: %v", e.Err)
}

func (e *FailedToCreateListError) Unwrap() error {
	return e.Err
}

type FailedToQueryListsError struct {
	Err error
}

func (e *FailedToQueryListsError) Error() string {
	return fmt.Sprintf("failed to query lists: %v", e.Err)
}

func (e *FailedToQueryListsError) Unwrap() error {
	return e.Err
}

type FailedToScanListError struct {
	Err error
}

func (e *FailedToScanListError) Error() string {
	return fmt.Sprintf("failed to scan list: %v", e.Err)
}

func (e *FailedToScanListError) Unwrap() error {
	return e.Err
}

type FailedToUpdateListError struct {
	Err error
}

func (e *FailedToUpdateListError) Error() string {
	return fmt.Sprintf("failed to update list: %v", e.Err)
}

func (e *FailedToUpdateListError) Unwrap() error {
	return e.Err
}

type FailedToDeleteListError struct {
	Err error
}

func (e *FailedToDeleteListError) Error() string {
	return fmt.Sprintf("failed to delete list: %v", e.Err)
}

func (e *FailedToDeleteListError) Unwrap() error {
	return e.Err
}

type FailedToGetListQuantitiesError struct {
	Err error
}

func (e *FailedToGetListQuantitiesError) Error() string {
	return fmt.Sprintf("failed to get list quantities: %v", e.Err)
}

func (e *FailedToGetListQuantitiesError) Unwrap() error {
	return e.Err
}

type FailedToUpsertListCardError struct {
	Err error
}

func (e *FailedToUpsertListCardError) Error() string {
	return fmt.Sprintf("failed to upsert list card: %v", e.Err)
}

func (e *FailedToUpsertListCardError) Unwrap() error {
	return e.Err
}
