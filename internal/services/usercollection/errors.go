package usercollection

import "fmt"

type FailedToQueryUserCollectionError struct {
	Err error
}

func (e *FailedToQueryUserCollectionError) Error() string {
	return fmt.Sprintf("failed to query user collection: %v", e.Err)
}

func (e *FailedToQueryUserCollectionError) Unwrap() error {
	return e.Err
}

type FailedToQueryUserCollectionByGameError struct {
	Err error
}

func (e *FailedToQueryUserCollectionByGameError) Error() string {
	return fmt.Sprintf("failed to query user collection by game: %v", e.Err)
}

func (e *FailedToQueryUserCollectionByGameError) Unwrap() error {
	return e.Err
}

type FailedToScanUserCollectionError struct {
	Err error
}

func (e *FailedToScanUserCollectionError) Error() string {
	return fmt.Sprintf("failed to scan user collection: %v", e.Err)
}

func (e *FailedToScanUserCollectionError) Unwrap() error {
	return e.Err
}

type ErrorIteratingUserCollectionsError struct {
	Err error
}

func (e *ErrorIteratingUserCollectionsError) Error() string {
	return fmt.Sprintf("error iterating user collections: %v", e.Err)
}

func (e *ErrorIteratingUserCollectionsError) Unwrap() error {
	return e.Err
}

type FailedToCreateSampleCollectionDataError struct {
	Err error
}

func (e *FailedToCreateSampleCollectionDataError) Error() string {
	return fmt.Sprintf("failed to create sample collection data: %v", e.Err)
}

func (e *FailedToCreateSampleCollectionDataError) Unwrap() error {
	return e.Err
}

type FailedToGetQuantityError struct {
	Err error
}

func (e *FailedToGetQuantityError) Error() string {
	return fmt.Sprintf("failed to get quantity: %v", e.Err)
}

func (e *FailedToGetQuantityError) Unwrap() error {
	return e.Err
}

type FailedToIncrementQuantityError struct {
	Err error
}

func (e *FailedToIncrementQuantityError) Error() string {
	return fmt.Sprintf("failed to increment quantity: %v", e.Err)
}

func (e *FailedToIncrementQuantityError) Unwrap() error {
	return e.Err
}

type FailedToDecrementQuantityError struct {
	Err error
}

func (e *FailedToDecrementQuantityError) Error() string {
	return fmt.Sprintf("failed to decrement quantity: %v", e.Err)
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
