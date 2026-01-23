package cli

import (
	"fmt"
	"io"

	"github.com/alexbrand/backlog/internal/output"
)

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
	Code     int
	JSONCode string // Optional specific error code for JSON output (e.g., "INVALID_INPUT")
	Message  string
	Err      error
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

// InvalidInputError creates an invalid input error (exit code 1).
func InvalidInputError(message string) *ExitCodeError {
	return &ExitCodeError{Code: ExitError, JSONCode: "INVALID_INPUT", Message: message}
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

// ExitCodeToString converts a numeric exit code to a string error code.
// These codes are used in JSON error output.
func ExitCodeToString(code int) string {
	switch code {
	case ExitSuccess:
		return "SUCCESS"
	case ExitError:
		return "ERROR"
	case ExitConflict:
		return "CONFLICT"
	case ExitNotFound:
		return "NOT_FOUND"
	case ExitConfigError:
		return "CONFIG_ERROR"
	default:
		return "ERROR"
	}
}

// GetJSONCode returns the appropriate JSON error code for an error.
// It first checks for a specific JSONCode on ExitCodeError, then falls back
// to converting the exit code.
func GetJSONCode(err error) string {
	if exitErr, ok := err.(*ExitCodeError); ok && exitErr.JSONCode != "" {
		return exitErr.JSONCode
	}
	return ExitCodeToString(GetExitCode(err))
}

// PrintError outputs an error using the appropriate formatter.
// When format is "json", it outputs a structured JSON error to the writer.
// For other formats, it outputs a plain text error message.
func PrintError(w io.Writer, err error, format string) {
	if err == nil {
		return
	}

	formatter := output.New(output.Format(format))
	codeStr := GetJSONCode(err)

	formatter.FormatError(w, codeStr, err.Error(), nil)
}
