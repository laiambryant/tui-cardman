package runtimecfg

import "fmt"

type LocalConfigLoadError struct {
	err error
}

func (l LocalConfigLoadError) Error() string {
	return fmt.Sprintf("failed to load local config: %v", l.err)
}

func (l LocalConfigLoadError) Unwrap() error {
	return l.err
}

type FailedToSaveConfigError struct {
	err error
}

func (f FailedToSaveConfigError) Error() string {
	return fmt.Sprintf("failed to save config: %v", f.err)
}

func (f FailedToSaveConfigError) Unwrap() error {
	return f.err
}

type NoRemoteServiceAvailable struct{}

func (n NoRemoteServiceAvailable) Error() string {
	return "no remote service available"
}
