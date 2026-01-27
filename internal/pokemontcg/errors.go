package pokemontcg

import "fmt"

type FailedToGetPokemonCardGameIDError struct {
	Err error
}

func (e *FailedToGetPokemonCardGameIDError) Error() string {
	return fmt.Sprintf("failed to get Pokemon card game ID: %v", e.Err)
}

func (e *FailedToGetPokemonCardGameIDError) Unwrap() error {
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

type FailedToCommitCardTransactionError struct {
	Err error
}

func (e *FailedToCommitCardTransactionError) Error() string {
	return fmt.Sprintf("failed to commit card transaction: %v", e.Err)
}

func (e *FailedToCommitCardTransactionError) Unwrap() error {
	return e.Err
}

type FailedToUpsertSetError struct {
	Err error
}

func (e *FailedToUpsertSetError) Error() string {
	return fmt.Sprintf("failed to upsert set: %v", e.Err)
}

func (e *FailedToUpsertSetError) Unwrap() error {
	return e.Err
}

type FailedToFetchCardsForSetError struct {
	SetID string
	Page  int
	Err   error
}

func (e *FailedToFetchCardsForSetError) Error() string {
	return fmt.Sprintf("failed to fetch cards for set %s page %d: %v", e.SetID, e.Page, e.Err)
}

func (e *FailedToFetchCardsForSetError) Unwrap() error {
	return e.Err
}

type FailedToCommitSetTransactionError struct {
	Err error
}

func (e *FailedToCommitSetTransactionError) Error() string {
	return fmt.Sprintf("failed to commit set transaction: %v", e.Err)
}

func (e *FailedToCommitSetTransactionError) Unwrap() error {
	return e.Err
}

type FailedToCreateImportRunError struct {
	Err error
}

func (e *FailedToCreateImportRunError) Error() string {
	return fmt.Sprintf("failed to create import run: %v", e.Err)
}

func (e *FailedToCreateImportRunError) Unwrap() error {
	return e.Err
}

type FailedToFetchSetsError struct {
	Err error
}

func (e *FailedToFetchSetsError) Error() string {
	return fmt.Sprintf("failed to fetch sets: %v", e.Err)
}

func (e *FailedToFetchSetsError) Unwrap() error {
	return e.Err
}

type FailedToUpdateImportRunError struct {
	Err error
}

func (e *FailedToUpdateImportRunError) Error() string {
	return fmt.Sprintf("failed to update import run: %v", e.Err)
}

func (e *FailedToUpdateImportRunError) Unwrap() error {
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

type ImportSetsNotFoundError struct {
	Message string
}

func (e *ImportSetsNotFoundError) Error() string {
	return e.Message
}

type RateLimiterError struct {
	Err error
}

func (e *RateLimiterError) Error() string {
	return fmt.Sprintf("rate limiter error: %v", e.Err)
}

func (e *RateLimiterError) Unwrap() error {
	return e.Err
}

type FailedToFetchCardsError struct {
	Err error
}

func (e *FailedToFetchCardsError) Error() string {
	return fmt.Sprintf("failed to fetch cards: %v", e.Err)
}

func (e *FailedToFetchCardsError) Unwrap() error {
	return e.Err
}

type FailedToFetchCardError struct {
	Err error
}

func (e *FailedToFetchCardError) Error() string {
	return fmt.Sprintf("failed to fetch card: %v", e.Err)
}

func (e *FailedToFetchCardError) Unwrap() error {
	return e.Err
}
