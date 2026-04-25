package promise

import "fmt"

// TypeError 类型错误，通常用于参数不允许为 nil 的情况
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

// RangeError 范围错误，通常用于数值类型参数不再允许范围的情况
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

// TimeoutError 超时错误，用于 [EventLoop.Await] 函数等待超时的情况
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

// AggregateError 聚合错误，通常用于批量处理 [Promise] 时某些条件不满足的情况
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

// UnexpectedError 非预期错误，用于拒绝理由不是 error 类型的情况，将其包装为 error 类型
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
