package card

import "fmt"

type FailedToQueryCardsByGameIDError struct {
	Err error
}

func (e *FailedToQueryCardsByGameIDError) Error() string {
	return fmt.Sprintf("failed to query cards by game ID: %v", e.Err)
}

func (e *FailedToQueryCardsByGameIDError) Unwrap() error {
	return e.Err
}

type FailedToQueryAllCardsError struct {
	Err error
}

func (e *FailedToQueryAllCardsError) Error() string {
	return fmt.Sprintf("failed to query all cards: %v", e.Err)
}

func (e *FailedToQueryAllCardsError) Unwrap() error {
	return e.Err
}

type FailedToScanCardError struct {
	Err error
}

func (e *FailedToScanCardError) Error() string {
	return fmt.Sprintf("failed to scan card: %v", e.Err)
}

func (e *FailedToScanCardError) Unwrap() error {
	return e.Err
}

type ErrorIteratingCardsError struct {
	Err error
}

func (e *ErrorIteratingCardsError) Error() string {
	return fmt.Sprintf("error iterating cards: %v", e.Err)
}

func (e *ErrorIteratingCardsError) Unwrap() error {
	return e.Err
}

type FailedToInsertCardError struct {
	Err error
}

func (e *FailedToInsertCardError) Error() string {
	return fmt.Sprintf("failed to insert card: %v", e.Err)
}

func (e *FailedToInsertCardError) Unwrap() error {
	return e.Err
}

type FailedToGetCardIDError struct {
	Err error
}

func (e *FailedToGetCardIDError) Error() string {
	return fmt.Sprintf("failed to get card ID: %v", e.Err)
}

func (e *FailedToGetCardIDError) Unwrap() error {
	return e.Err
}

type FailedToQueryCardError struct {
	Err error
}

func (e *FailedToQueryCardError) Error() string {
	return fmt.Sprintf("failed to query card: %v", e.Err)
}

func (e *FailedToQueryCardError) Unwrap() error {
	return e.Err
}

type FailedToUpdateCardError struct {
	Err error
}

func (e *FailedToUpdateCardError) Error() string {
	return fmt.Sprintf("failed to update card: %v", e.Err)
}

func (e *FailedToUpdateCardError) Unwrap() error {
	return e.Err
}
