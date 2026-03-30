package mediaforge

import (
	"errors"
	"fmt"
)

// ErrorCategory classifies the type of error.
type ErrorCategory string

const (
	CategoryConfig     ErrorCategory = "config"
	CategoryValidation ErrorCategory = "validation"
	CategoryProcessing ErrorCategory = "processing"
	CategoryStorage    ErrorCategory = "storage"
	CategorySigning    ErrorCategory = "signing"
)

// Error is the library's structured error type.
type Error struct {
	Category  ErrorCategory
	Msg       string
	Retryable bool
	Op        string
	Detail    string
	err       error
}

func (e *Error) Error() string {
	s := fmt.Sprintf("mediastore [%s]: %s", e.Category, e.Msg)
	if e.Op != "" {
		s = fmt.Sprintf("mediastore [%s] %s: %s", e.Category, e.Op, e.Msg)
	}
	if e.Detail != "" {
		s += " (" + e.Detail + ")"
	}
	if e.err != nil {
		s += ": " + e.err.Error()
	}
	return s
}

func (e *Error) Unwrap() error {
	return e.err
}

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Category == t.Category
}

// WithOp sets the operation name.
func (e *Error) WithOp(op string) *Error {
	e.Op = op
	return e
}

// WithDetail sets additional technical detail.
func (e *Error) WithDetail(detail string) *Error {
	e.Detail = detail
	return e
}

// WithCause wraps an underlying error.
func (e *Error) WithCause(err error) *Error {
	e.err = err
	return e
}

// ErrConfig creates a config category error.
func ErrConfig(msg string) *Error {
	return &Error{Category: CategoryConfig, Msg: msg, Retryable: false}
}

// ErrValidation creates a validation category error.
func ErrValidation(msg string) *Error {
	return &Error{Category: CategoryValidation, Msg: msg, Retryable: false}
}

// ErrProcessing creates a processing category error.
func ErrProcessing(msg string) *Error {
	return &Error{Category: CategoryProcessing, Msg: msg, Retryable: false}
}

// ErrStorage creates a storage category error.
func ErrStorage(msg string, retryable bool) *Error {
	return &Error{Category: CategoryStorage, Msg: msg, Retryable: retryable}
}

// ErrSigning creates a signing category error.
func ErrSigning(msg string) *Error {
	return &Error{Category: CategorySigning, Msg: msg, Retryable: false}
}

// IsRetryable checks if any error in the chain is retryable.
// It walks the chain using errors.Unwrap so every node is visited exactly once.
func IsRetryable(err error) bool {
	for err != nil {
		var e *Error
		if errors.As(err, &e) {
			if e.Retryable {
				return true
			}
			// Advance past this *Error node to avoid revisiting it.
			err = e.Unwrap()
			continue
		}
		err = errors.Unwrap(err)
	}
	return false
}
