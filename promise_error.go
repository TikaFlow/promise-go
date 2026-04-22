package promise

import "fmt"

type TypeError struct {
	error
	text string
}

func (e *TypeError) Error() string {
	return "TypeError: " + e.text
}

func NewTypeError(text string) *TypeError {
	return &TypeError{text: text}
}

type RangeError struct {
	error
	text string
}

func (e *RangeError) Error() string {
	return "RangeError: " + e.text
}

func NewRangeError(text string) *RangeError {
	return &RangeError{text: text}
}

type TimeoutError struct {
	error
	text string
}

func (e *TimeoutError) Error() string {
	return "TimeoutError: " + e.text
}

func NewTimeoutError(text string) *TimeoutError {
	return &TimeoutError{text: text}
}

type AggregateError struct {
	error
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

type UnexpectedError struct {
	error
	reason any
}

func (e *UnexpectedError) Error() string {
	return "UnexpectedError: " + fmt.Sprintf("%v", e.reason)
}

func NewUnexpectedError(reason any) *UnexpectedError {
	return &UnexpectedError{reason: reason}
}
