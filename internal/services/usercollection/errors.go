package usercollection

import "fmt"

type FailedToGetQuantityError struct {
	Err error
}

func (e *FailedToGetQuantityError) Error() string {
	return fmt.Sprintf("failed to get card quantity: %v", e.Err)
}
func (e *FailedToGetQuantityError) Unwrap() error {
	return e.Err
}

type FailedToIncrementQuantityError struct {
	Err error
}

func (e *FailedToIncrementQuantityError) Error() string {
	return fmt.Sprintf("failed to increment card quantity: %v", e.Err)
}
func (e *FailedToIncrementQuantityError) Unwrap() error {
	return e.Err
}

type FailedToDecrementQuantityError struct {
	Err error
}

func (e *FailedToDecrementQuantityError) Error() string {
	return fmt.Sprintf("failed to decrement card quantity: %v", e.Err)
}
func (e *FailedToDecrementQuantityError) Unwrap() error {
	return e.Err
}

type FailedToUpsertCollectionError struct {
	Err error
}

func (e *FailedToUpsertCollectionError) Error() string {
	return fmt.Sprintf("failed to upsert collection: %v", e.Err)
}
func (e *FailedToUpsertCollectionError) Unwrap() error {
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
