package buttonconfig

import "fmt"

type FailedToMarshalConfigurationError struct {
	Err error
}

func (e *FailedToMarshalConfigurationError) Error() string {
	return fmt.Sprintf("failed to marshal configuration: %v", e.Err)
}

func (e *FailedToMarshalConfigurationError) Unwrap() error {
	return e.Err
}

type FailedToSaveButtonConfigurationError struct {
	Err error
}

func (e *FailedToSaveButtonConfigurationError) Error() string {
	return fmt.Sprintf("failed to save button configuration: %v", e.Err)
}

func (e *FailedToSaveButtonConfigurationError) Unwrap() error {
	return e.Err
}

type FailedToGetRowsAffectedError struct {
	Err error
}

func (e *FailedToGetRowsAffectedError) Error() string {
	return fmt.Sprintf("failed to get rows affected: %v", e.Err)
}

func (e *FailedToGetRowsAffectedError) Unwrap() error {
	return e.Err
}

type FailedToCheckExistingConfigError struct {
	Err error
}

func (e *FailedToCheckExistingConfigError) Error() string {
	return fmt.Sprintf("failed to check existing config: %v", e.Err)
}

func (e *FailedToCheckExistingConfigError) Unwrap() error {
	return e.Err
}

type FailedToLoadLocalConfigForMigrationError struct {
	Err error
}

func (e *FailedToLoadLocalConfigForMigrationError) Error() string {
	return fmt.Sprintf("failed to load local config for migration: %v", e.Err)
}

func (e *FailedToLoadLocalConfigForMigrationError) Unwrap() error {
	return e.Err
}

type FailedToMigrateConfigToDatabaseError struct {
	Err error
}

func (e *FailedToMigrateConfigToDatabaseError) Error() string {
	return fmt.Sprintf("failed to migrate config to database: %v", e.Err)
}

func (e *FailedToMigrateConfigToDatabaseError) Unwrap() error {
	return e.Err
}

type FailedToGetButtonConfigurationError struct {
	Err error
}

func (e *FailedToGetButtonConfigurationError) Error() string {
	return fmt.Sprintf("failed to get button configuration: %v", e.Err)
}

func (e *FailedToGetButtonConfigurationError) Unwrap() error {
	return e.Err
}
