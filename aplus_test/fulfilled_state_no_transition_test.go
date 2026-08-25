package aplus_test

import (
	"sync/atomic"
	"testing"

	. "github.com/TikaFlow/promise-go"
)

// 2.1.2: 当已 fulfilled 后，promise 不得再迁移到任何其他状态。
func TestAplus2_1_2(t *testing.T) {
	t.Parallel()
	testFulfilled(t, dummy, func(t *testing.T, p *Promise) *Promise {
		return p.Then(func(v any) (any, error) { return nil, nil },
			func(r error) (any, error) {
				t.Errorf("promise transitioned to rejected after being fulfilled: %v", r)
				return nil, nil
			})
	})

	t.Run("trying to fulfill then immediately reject", func(t *testing.T) {
		t.Parallel()
		// adjusted: resolve then reject（reject 因 sync.Once 被忽略）
		fulfillThenReject(t, func(resolve, reject func(any)) {
			resolve(dummy)
			reject(dummy)
		})
	})

	t.Run("trying to fulfill then reject, delayed", func(t *testing.T) {
		t.Parallel()
		fulfillThenReject(t, func(resolve, reject func(any)) {
			el.SetTimeout(func() {
				resolve(dummy)
				reject(dummy)
			}, 50)
		})
	})

	t.Run("trying to fulfill immediately then reject delayed", func(t *testing.T) {
		t.Parallel()
		fulfillThenReject(t, func(resolve, reject func(any)) {
			resolve(dummy)
			el.SetTimeout(func() { reject(dummy) }, 50)
		})
	})
}

// fulfillThenReject 断言：先 fulfill 后 reject 时，onRejected 不触发且 onFulfilled 恰触发一次。
func fulfillThenReject(t *testing.T, trigger func(resolve, reject func(any))) {
	t.Helper()
	p, resolve, reject := deferred()
	var onFulfilledCalled atomic.Bool

	tail := p.Then(func(v any) (any, error) {
		onFulfilledCalled.Store(true)
		return nil, nil
	}, func(r error) (any, error) {
		t.Errorf("onRejected should not fire after fulfillment, got %v", r)
		return nil, nil
	})

	trigger(resolve, reject)
	waitTail(t, tail)

	if !onFulfilledCalled.Load() {
		t.Errorf("onFulfilled should have been called once")
	}
}
