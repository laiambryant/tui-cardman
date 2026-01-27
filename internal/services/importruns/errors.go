package importruns

import "fmt"

type FailedToCreateImportRunError struct {
	Err error
}

func (e *FailedToCreateImportRunError) Error() string {
	return fmt.Sprintf("failed to create import run: %v", e.Err)
}

func (e *FailedToCreateImportRunError) Unwrap() error {
	return e.Err
}

type FailedToGetLastInsertIDError struct {
	Err error
}

func (e *FailedToGetLastInsertIDError) Error() string {
	return fmt.Sprintf("failed to get last insert id: %v", e.Err)
}

func (e *FailedToGetLastInsertIDError) Unwrap() error {
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
