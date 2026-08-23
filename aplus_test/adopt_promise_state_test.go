package aplus_test

import (
	"sync/atomic"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 2.3.2: 若 x 是 promise，则采纳其状态（pending / fulfilled / rejected）。
func TestAplus2_3_2(t *testing.T) {
	t.Run("2.3.2.1 x is pending, promise stays pending", func(t *testing.T) {
		testPromiseResolution(t, func() any {
			p, _, _ := deferred()
			return p
		}, func(t *testing.T, promise *Promise) *Promise {
			var wasFulfilled, wasRejected atomic.Bool
			promise.Then(func(v any) (any, error) {
				wasFulfilled.Store(true)
				return nil, nil
			}, func(r error) (any, error) {
				wasRejected.Store(true)
				return nil, nil
			})

			time.Sleep(100 * time.Millisecond)
			if wasFulfilled.Load() || wasRejected.Load() {
				t.Errorf("promise should remain pending while x is pending, got fulfilled=%v rejected=%v",
					wasFulfilled.Load(), wasRejected.Load())
			}
			return nil
		})
	})

	t.Run("2.3.2.2 x is fulfilled, adopt the same value", func(t *testing.T) {
		t.Run("x is already-fulfilled", func(t *testing.T) {
			testPromiseResolution(t, func() any { return el.Resolve(sentinel) },
				func(t *testing.T, promise *Promise) *Promise {
					return expectFulfilled(t, promise, sentinel)
				})
		})
		t.Run("x is eventually-fulfilled", func(t *testing.T) {
			testPromiseResolution(t, func() any {
				p, resolve, _ := deferred()
				el.SetTimeout(func() { resolve(sentinel) }, 50)
				return p
			}, func(t *testing.T, promise *Promise) *Promise {
				return expectFulfilled(t, promise, sentinel)
			})
		})
	})

	t.Run("2.3.2.3 x is rejected, adopt the same reason", func(t *testing.T) {
		t.Run("x is already-rejected", func(t *testing.T) {
			testPromiseResolution(t, func() any { return el.Reject(sentinel) },
				func(t *testing.T, promise *Promise) *Promise {
					return expectRejected(t, promise, sentinel)
				})
		})
		t.Run("x is eventually-rejected", func(t *testing.T) {
			testPromiseResolution(t, func() any {
				p, _, reject := deferred()
				el.SetTimeout(func() { reject(sentinel) }, 50)
				return p
			}, func(t *testing.T, promise *Promise) *Promise {
				return expectRejected(t, promise, sentinel)
			})
		})
	})
}
