package runtimecfg

import "fmt"

type FailedToReadConfigFileError struct {
	Err error
}

func (e *FailedToReadConfigFileError) Error() string {
	return fmt.Sprintf("failed to read config file: %v", e.Err)
}

func (e *FailedToReadConfigFileError) Unwrap() error {
	return e.Err
}

type FailedToParseConfigFileError struct {
	Err error
}

func (e *FailedToParseConfigFileError) Error() string {
	return fmt.Sprintf("failed to parse config file: %v", e.Err)
}

func (e *FailedToParseConfigFileError) Unwrap() error {
	return e.Err
}

type FailedToCreateConfigDirectoryError struct {
	Err error
}

func (e *FailedToCreateConfigDirectoryError) Error() string {
	return fmt.Sprintf("failed to create config directory: %v", e.Err)
}

func (e *FailedToCreateConfigDirectoryError) Unwrap() error {
	return e.Err
}

type FailedToMarshalConfigError struct {
	Err error
}

func (e *FailedToMarshalConfigError) Error() string {
	return fmt.Sprintf("failed to marshal config: %v", e.Err)
}

func (e *FailedToMarshalConfigError) Unwrap() error {
	return e.Err
}

type FailedToWriteConfigFileError struct {
	Err error
}

func (e *FailedToWriteConfigFileError) Error() string {
	return fmt.Sprintf("failed to write config file: %v", e.Err)
}

func (e *FailedToWriteConfigFileError) Unwrap() error {
	return e.Err
}

type ActionNotBoundToAnyKeyError struct {
	Action string
}

func (e *ActionNotBoundToAnyKeyError) Error() string {
	return fmt.Sprintf("action '%s' is not bound to any key", e.Action)
}

type KeyBoundToMultipleActionsError struct {
	Key            string
	ExistingAction string
	NewAction      string
}

func (e *KeyBoundToMultipleActionsError) Error() string {
	return fmt.Sprintf("key '%s' is bound to both '%s' and '%s'", e.Key, e.ExistingAction, e.NewAction)
}

type ThemeDirectoryNotFoundError struct {
	Path string
}

func (e *ThemeDirectoryNotFoundError) Error() string {
	return fmt.Sprintf("theme directory not found: %s", e.Path)
}

type FailedToReadThemeFileError struct {
	Path string
	Err  error
}

func (e *FailedToReadThemeFileError) Error() string {
	return fmt.Sprintf("failed to read theme file %s: %v", e.Path, e.Err)
}

func (e *FailedToReadThemeFileError) Unwrap() error {
	return e.Err
}

type FailedToParseThemeFileError struct {
	Path string
	Err  error
}

func (e *FailedToParseThemeFileError) Error() string {
	return fmt.Sprintf("failed to parse theme file %s: %v", e.Path, e.Err)
}

func (e *FailedToParseThemeFileError) Unwrap() error {
	return e.Err
}

type InvalidThemeFormatError struct {
	Path  string
	Field string
}

func (e *InvalidThemeFormatError) Error() string {
	return fmt.Sprintf("invalid theme format in %s: missing or invalid field '%s'", e.Path, e.Field)
}
