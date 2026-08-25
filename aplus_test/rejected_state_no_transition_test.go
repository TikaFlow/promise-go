package aplus_test

import (
	"sync/atomic"
	"testing"

	. "github.com/TikaFlow/promise-go"
)

// 2.1.3: 当已 rejected 后，promise 不得再迁移到任何其他状态。
func TestAplus2_1_3(t *testing.T) {
	t.Parallel()
	testRejected(t, dummy, func(t *testing.T, p *Promise) *Promise {
		return p.Then(func(v any) (any, error) {
			t.Errorf("promise transitioned to fulfilled after being rejected: %#v", v)
			return nil, nil
		}, func(r error) (any, error) {
			return nil, nil
		})
	})

	t.Run("trying to reject then immediately fulfill", func(t *testing.T) {
		t.Parallel()
		rejectThenFulfill(t, func(resolve, reject func(any)) {
			reject(dummy)
			resolve(dummy)
		})
	})

	t.Run("trying to reject then fulfill, delayed", func(t *testing.T) {
		t.Parallel()
		rejectThenFulfill(t, func(resolve, reject func(any)) {
			el.SetTimeout(func() {
				reject(dummy)
				resolve(dummy)
			}, 50)
		})
	})

	t.Run("trying to reject immediately then fulfill delayed", func(t *testing.T) {
		t.Parallel()
		rejectThenFulfill(t, func(resolve, reject func(any)) {
			reject(dummy)
			el.SetTimeout(func() { resolve(dummy) }, 50)
		})
	})
}

// rejectThenFulfill 断言：先 reject 后 resolve 时，onFulfilled 不触发且 onRejected 恰触发一次。
func rejectThenFulfill(t *testing.T, trigger func(resolve, reject func(any))) {
	t.Helper()
	p, resolve, reject := deferred()
	var onRejectedCalled atomic.Bool

	tail := p.Then(func(v any) (any, error) {
		t.Errorf("onFulfilled should not fire after rejection, got %#v", v)
		return nil, nil
	}, func(r error) (any, error) {
		onRejectedCalled.Store(true)
		return nil, nil
	})

	trigger(resolve, reject)
	waitTail(t, tail)

	if !onRejectedCalled.Load() {
		t.Errorf("onRejected should have been called once")
	}
}
