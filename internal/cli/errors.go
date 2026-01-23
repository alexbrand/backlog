package cli

import "fmt"

// Exit codes as defined in the PRD
const (
	ExitSuccess      = 0
	ExitError        = 1 // General error (network, auth, invalid input)
	ExitConflict     = 2 // Conflict (task already claimed, state conflict)
	ExitNotFound     = 3 // Not found (task doesn't exist)
	ExitConfigError  = 4 // Configuration error
)

// ExitError is an error that carries an exit code.
type ExitCodeError struct {
	Code    int
	Message string
	Err     error
}

func (e *ExitCodeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *ExitCodeError) Unwrap() error {
	return e.Err
}

// NewExitCodeError creates a new ExitCodeError with the given code and message.
func NewExitCodeError(code int, message string) *ExitCodeError {
	return &ExitCodeError{Code: code, Message: message}
}

// WrapExitCodeError wraps an existing error with an exit code.
func WrapExitCodeError(code int, message string, err error) *ExitCodeError {
	return &ExitCodeError{Code: code, Message: message, Err: err}
}

// NotFoundError creates a not found error (exit code 3).
func NotFoundError(message string) *ExitCodeError {
	return NewExitCodeError(ExitNotFound, message)
}

// ConflictError creates a conflict error (exit code 2).
func ConflictError(message string) *ExitCodeError {
	return NewExitCodeError(ExitConflict, message)
}

// ConfigError creates a configuration error (exit code 4).
func ConfigError(message string) *ExitCodeError {
	return NewExitCodeError(ExitConfigError, message)
}

// GetExitCode returns the exit code from an error.
// If the error is an ExitCodeError, returns its code.
// Otherwise, returns 1 (general error).
func GetExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	if exitErr, ok := err.(*ExitCodeError); ok {
		return exitErr.Code
	}
	return ExitError
}
