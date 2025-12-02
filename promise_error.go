package promise

import "fmt"

// TypeError
type TypeError struct {
	text string
}

func (e *TypeError) Error() string {
	return "TypeError: " + e.text
}

func NewTypeError(text string) *TypeError {
	return &TypeError{text}
}

// RangeError
type RangeError struct {
	text string
}

func (e *RangeError) Error() string {
	return "RangeError: " + e.text
}

func NewRangeError(text string) *RangeError {
	return &RangeError{text}
}

// NewTimeoutError
type TimeoutError struct {
	text string
}

func (e *TimeoutError) Error() string {
	return "TimeoutError: " + e.text
}

func NewTimeoutError(text string) *TimeoutError {
	return &TimeoutError{text}
}

// AggregateError
type AggregateError struct {
	errors  []error
	stack   string
	message string
}

func (e *AggregateError) Error() string {
	return "AggregateError: " + e.stack
}

func (e *AggregateError) Unwrap() []error {
	return e.errors
}

func NewAggregateError(errors []error, stack string, message string) *AggregateError {
	return &AggregateError{
		errors:  errors,
		stack:   stack,
		message: message,
	}
}

// UnexpectedError
type UnexpectedError struct {
	reason any
}

func (e *UnexpectedError) Error() string {
	return "UnexpectedError: " + fmt.Sprintf("%v", e.reason)
}

func NewUnexpectedError(reason any) *UnexpectedError {
	return &UnexpectedError{reason}
}
