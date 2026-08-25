package aplus_test

import (
	"sync/atomic"
	"testing"
)

// 2.2.1: onFulfilled / onRejected 均为可选参数，非函数则忽略（Go 中只能用 nil 表示“非函数”）。
func TestAplus2_2_1(t *testing.T) {
	t.Parallel()
	t.Run("2.2.1.1 onFulfilled not a function, directly-rejected promise", func(t *testing.T) {
		t.Parallel()
		var onRejectedCalled atomic.Bool
		tail := rejected(dummy).Then(nil, func(r error) (any, error) {
			onRejectedCalled.Store(true)
			return nil, nil
		})
		waitTail(t, tail)
		if !onRejectedCalled.Load() {
			t.Errorf("onRejected should fire when onFulfilled is nil (ignored)")
		}
	})

	t.Run("2.2.1.1 onFulfilled not a function, rejected then chained off", func(t *testing.T) {
		t.Parallel()
		p := rejected(dummy).Then(func(any) (any, error) { return nil, nil }, nil)
		var onRejectedCalled atomic.Bool
		tail := p.Then(nil, func(r error) (any, error) {
			onRejectedCalled.Store(true)
			return nil, nil
		})
		waitTail(t, tail)
		if !onRejectedCalled.Load() {
			t.Errorf("onRejected should fire after chaining past a nil onFulfilled")
		}
	})

	t.Run("2.2.1.2 onRejected not a function, directly-fulfilled promise", func(t *testing.T) {
		t.Parallel()
		var onFulfilledCalled atomic.Bool
		tail := resolved(dummy).Then(func(v any) (any, error) {
			onFulfilledCalled.Store(true)
			return nil, nil
		}, nil)
		waitTail(t, tail)
		if !onFulfilledCalled.Load() {
			t.Errorf("onFulfilled should fire when onRejected is nil (ignored)")
		}
	})

	t.Run("2.2.1.2 onRejected not a function, fulfilled then chained off", func(t *testing.T) {
		t.Parallel()
		p := resolved(dummy).Then(nil, func(r error) (any, error) { return nil, nil })
		var onFulfilledCalled atomic.Bool
		tail := p.Then(func(v any) (any, error) {
			onFulfilledCalled.Store(true)
			return nil, nil
		}, nil)
		waitTail(t, tail)
		if !onFulfilledCalled.Load() {
			t.Errorf("onFulfilled should fire after chaining past a nil onRejected")
		}
	})

	skipNA(t, "2.2.1 other non-function handler values", 16,
		"false/5/object/array 等非函数取值在 Go 无对应物（Go 的 ThenCallback 只能为 func 或 nil）")
}
