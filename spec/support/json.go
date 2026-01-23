// Package support provides test helpers and fixtures for the backlog CLI specs.
package support

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// JSONResult wraps parsed JSON output for structured assertions.
type JSONResult struct {
	// Data holds the parsed JSON data
	Data any
	// Raw is the original JSON string
	Raw string
	// ParseErr is set if JSON parsing failed
	ParseErr error
}

// ParseJSON parses a JSON string and returns a JSONResult for assertions.
func ParseJSON(jsonStr string) *JSONResult {
	result := &JSONResult{
		Raw: jsonStr,
	}

	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		result.ParseErr = err
		return result
	}

	result.Data = data
	return result
}

// ParseJSONFromResult parses the stdout of a CommandResult as JSON.
func ParseJSONFromResult(cmdResult *CommandResult) *JSONResult {
	return ParseJSON(cmdResult.Stdout)
}

// Valid returns true if the JSON was parsed successfully.
func (r *JSONResult) Valid() bool {
	return r.ParseErr == nil
}

// Error returns the parse error message, or empty string if valid.
func (r *JSONResult) Error() string {
	if r.ParseErr == nil {
		return ""
	}
	return r.ParseErr.Error()
}

// Get retrieves a value at the given path using dot notation.
// Supports array indexing with brackets: "tasks[0].id"
// Returns nil if the path doesn't exist.
func (r *JSONResult) Get(path string) any {
	if r.ParseErr != nil || r.Data == nil {
		return nil
	}
	return getPath(r.Data, path)
}

// GetString retrieves a string value at the given path.
// Returns empty string if not found or not a string.
func (r *JSONResult) GetString(path string) string {
	val := r.Get(path)
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// GetInt retrieves an integer value at the given path.
// Returns 0 if not found or not a number.
func (r *JSONResult) GetInt(path string) int {
	val := r.Get(path)
	if val == nil {
		return 0
	}
	// JSON numbers are float64 by default
	if f, ok := val.(float64); ok {
		return int(f)
	}
	return 0
}

// GetFloat retrieves a float value at the given path.
// Returns 0 if not found or not a number.
func (r *JSONResult) GetFloat(path string) float64 {
	val := r.Get(path)
	if val == nil {
		return 0
	}
	if f, ok := val.(float64); ok {
		return f
	}
	return 0
}

// GetBool retrieves a boolean value at the given path.
// Returns false if not found or not a boolean.
func (r *JSONResult) GetBool(path string) bool {
	val := r.Get(path)
	if val == nil {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

// GetArray retrieves an array at the given path.
// Returns nil if not found or not an array.
func (r *JSONResult) GetArray(path string) []any {
	val := r.Get(path)
	if val == nil {
		return nil
	}
	if arr, ok := val.([]any); ok {
		return arr
	}
	return nil
}

// GetObject retrieves an object at the given path.
// Returns nil if not found or not an object.
func (r *JSONResult) GetObject(path string) map[string]any {
	val := r.Get(path)
	if val == nil {
		return nil
	}
	if obj, ok := val.(map[string]any); ok {
		return obj
	}
	return nil
}

// ArrayLen returns the length of an array at the given path.
// Returns -1 if not found or not an array.
func (r *JSONResult) ArrayLen(path string) int {
	arr := r.GetArray(path)
	if arr == nil {
		return -1
	}
	return len(arr)
}

// Has returns true if a value exists at the given path (even if null).
func (r *JSONResult) Has(path string) bool {
	if r.ParseErr != nil || r.Data == nil {
		return false
	}
	return hasPath(r.Data, path)
}

// Equals checks if the value at path equals the expected value.
func (r *JSONResult) Equals(path string, expected any) bool {
	val := r.Get(path)
	return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", expected)
}

// StringEquals checks if the string value at path equals the expected string.
func (r *JSONResult) StringEquals(path, expected string) bool {
	return r.GetString(path) == expected
}

// IntEquals checks if the int value at path equals the expected int.
func (r *JSONResult) IntEquals(path string, expected int) bool {
	return r.GetInt(path) == expected
}

// BoolEquals checks if the bool value at path equals the expected bool.
func (r *JSONResult) BoolEquals(path string, expected bool) bool {
	return r.GetBool(path) == expected
}

// Contains checks if an array at path contains the expected value.
func (r *JSONResult) Contains(path string, expected any) bool {
	arr := r.GetArray(path)
	if arr == nil {
		return false
	}
	expectedStr := fmt.Sprintf("%v", expected)
	for _, item := range arr {
		if fmt.Sprintf("%v", item) == expectedStr {
			return true
		}
	}
	return false
}

// ContainsString checks if an array at path contains the expected string.
func (r *JSONResult) ContainsString(path, expected string) bool {
	arr := r.GetArray(path)
	if arr == nil {
		return false
	}
	for _, item := range arr {
		if s, ok := item.(string); ok && s == expected {
			return true
		}
	}
	return false
}

// IsNull returns true if the value at path is null.
func (r *JSONResult) IsNull(path string) bool {
	if !r.Has(path) {
		return false
	}
	return r.Get(path) == nil
}

// IsArray returns true if the value at path is an array.
func (r *JSONResult) IsArray(path string) bool {
	val := r.Get(path)
	_, ok := val.([]any)
	return ok
}

// IsObject returns true if the value at path is an object.
func (r *JSONResult) IsObject(path string) bool {
	val := r.Get(path)
	_, ok := val.(map[string]any)
	return ok
}

// getPath navigates to a value using dot notation with array support.
// Example paths: "tasks", "tasks[0]", "tasks[0].id", "error.code"
func getPath(data any, path string) any {
	if path == "" {
		return data
	}

	parts := parsePath(path)
	current := data

	for _, part := range parts {
		if current == nil {
			return nil
		}

		// Handle array index
		if idx, isIndex := parseArrayIndex(part); isIndex {
			if arr, ok := current.([]any); ok {
				if idx >= 0 && idx < len(arr) {
					current = arr[idx]
				} else {
					return nil
				}
			} else {
				return nil
			}
			continue
		}

		// Handle object key
		if obj, ok := current.(map[string]any); ok {
			current = obj[part]
		} else {
			return nil
		}
	}

	return current
}

// hasPath checks if a path exists in the data structure.
func hasPath(data any, path string) bool {
	if path == "" {
		return true
	}

	parts := parsePath(path)
	current := data

	for _, part := range parts {
		if current == nil {
			return false
		}

		// Handle array index
		if idx, isIndex := parseArrayIndex(part); isIndex {
			if arr, ok := current.([]any); ok {
				if idx >= 0 && idx < len(arr) {
					current = arr[idx]
				} else {
					return false
				}
			} else {
				return false
			}
			continue
		}

		// Handle object key
		if obj, ok := current.(map[string]any); ok {
			if _, exists := obj[part]; exists {
				current = obj[part]
			} else {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

// parsePath splits a path into parts, handling array notation.
// "tasks[0].id" -> ["tasks", "[0]", "id"]
func parsePath(path string) []string {
	var parts []string
	var current strings.Builder

	for i := 0; i < len(path); i++ {
		ch := path[i]
		switch ch {
		case '.':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		case '[':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			// Find matching ]
			end := strings.Index(path[i:], "]")
			if end == -1 {
				// Malformed, just add the rest
				current.WriteByte(ch)
			} else {
				parts = append(parts, path[i:i+end+1])
				i += end
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// parseArrayIndex parses "[N]" and returns the index and true if valid.
func parseArrayIndex(part string) (int, bool) {
	if len(part) < 3 || part[0] != '[' || part[len(part)-1] != ']' {
		return 0, false
	}
	idx, err := strconv.Atoi(part[1 : len(part)-1])
	if err != nil {
		return 0, false
	}
	return idx, true
}
