package sets

import "fmt"

type FailedToBeginTransactionError struct {
	Err error
}

func (e *FailedToBeginTransactionError) Error() string {
	return fmt.Sprintf("failed to begin transaction: %v", e.Err)
}

func (e *FailedToBeginTransactionError) Unwrap() error {
	return e.Err
}

type FailedToInsertSetError struct {
	Err error
}

func (e *FailedToInsertSetError) Error() string {
	return fmt.Sprintf("failed to insert set: %v", e.Err)
}

func (e *FailedToInsertSetError) Unwrap() error {
	return e.Err
}

type FailedToGetLastInsertIDError struct {
	Err error
}

func (e *FailedToGetLastInsertIDError) Error() string {
	return fmt.Sprintf("failed to get last insert ID: %v", e.Err)
}

func (e *FailedToGetLastInsertIDError) Unwrap() error {
	return e.Err
}

type FailedToQuerySetError struct {
	Err error
}

func (e *FailedToQuerySetError) Error() string {
	return fmt.Sprintf("failed to query set: %v", e.Err)
}

func (e *FailedToQuerySetError) Unwrap() error {
	return e.Err
}

type FailedToUpdateSetError struct {
	Err error
}

func (e *FailedToUpdateSetError) Error() string {
	return fmt.Sprintf("failed to update set: %v", e.Err)
}

func (e *FailedToUpdateSetError) Unwrap() error {
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

type FailedToQueryExistingSetsError struct {
	Err error
}

func (e *FailedToQueryExistingSetsError) Error() string {
	return fmt.Sprintf("failed to query existing sets: %v", e.Err)
}

func (e *FailedToQueryExistingSetsError) Unwrap() error {
	return e.Err
}

type FailedToScanSetAPIIDError struct {
	Err error
}

func (e *FailedToScanSetAPIIDError) Error() string {
	return fmt.Sprintf("failed to scan set api_id: %v", e.Err)
}

func (e *FailedToScanSetAPIIDError) Unwrap() error {
	return e.Err
}

type ErrorIteratingSetsError struct {
	Err error
}

func (e *ErrorIteratingSetsError) Error() string {
	return fmt.Sprintf("error iterating sets: %v", e.Err)
}

func (e *ErrorIteratingSetsError) Unwrap() error {
	return e.Err
}

type FailedToCheckSetCollectionsError struct {
	Err error
}

func (e *FailedToCheckSetCollectionsError) Error() string {
	return fmt.Sprintf("failed to check set collections: %v", e.Err)
}

func (e *FailedToCheckSetCollectionsError) Unwrap() error {
	return e.Err
}
