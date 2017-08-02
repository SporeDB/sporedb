// Package operations contains database operations logic internals.
package operations

import "errors"

// Runner is the prototype used by operations.
// The current value may be modified by the runner and must be used as its output.
type Runner func(input []byte, current *Value) error

// Errors returned when an operation does not match stored datatype.
var (
	ErrNotNumeric  = errors.New("non-numeric value")
	ErrNotValidSet = errors.New("non-valid set")
)
