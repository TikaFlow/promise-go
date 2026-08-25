package aplus_test

import (
	"sync/atomic"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 2.2.2: 若 onFulfilled 是函数，则在 promise fulfilled 后以其值为参调用。
func TestAplus2_2_2(t *testing.T) {
	t.Parallel()
	// 2.2.2.1 with fulfill value
	testFulfilled(t, sentinel, func(t *testing.T, p *Promise) *Promise {
		return expectFulfilled(t, p, sentinel)
	})

	// 2.2.2.2 not before fulfillment
	t.Run("2.2.2.2 fulfilled after a delay", func(t *testing.T) {
		t.Parallel()
		p, resolve, _ := deferred()
		isFulfilled := false
		tail := p.Then(func(v any) (any, error) {
			if !isFulfilled {
				t.Errorf("onFulfilled called before promise was fulfilled")
			}
			return nil, nil
		}, nil)
		el.SetTimeout(func() {
			isFulfilled = true
			resolve(dummy)
		}, 50)
		waitTail(t, tail)
	})

	t.Run("2.2.2.2 never fulfilled", func(t *testing.T) {
		t.Parallel()
		p, _, _ := deferred()
		var onFulfilledCalled atomic.Bool
		p.Then(func(v any) (any, error) {
			onFulfilledCalled.Store(true)
			return nil, nil
		}, nil)
		time.Sleep(150 * time.Millisecond)
		if onFulfilledCalled.Load() {
			t.Errorf("onFulfilled must not be called if the promise is never fulfilled")
		}
	})

	// 2.2.2.3 not more than once
	t.Run("2.2.2.3 already-fulfilled", func(t *testing.T) {
		t.Parallel()
		times := 0
		tail := resolved(dummy).Then(func(v any) (any, error) {
			times++
			if times != 1 {
				t.Errorf("onFulfilled called %d times, want 1", times)
			}
			return nil, nil
		}, nil)
		waitTail(t, tail)
	})

	t.Run("2.2.2.3 fulfill pending more than once, immediately", func(t *testing.T) {
		t.Parallel()
		p, resolve, _ := deferred()
		times := 0
		tail := p.Then(func(v any) (any, error) {
			times++
			if times != 1 {
				t.Errorf("onFulfilled called %d times, want 1", times)
			}
			return nil, nil
		}, nil)
		resolve(dummy)
		resolve(dummy)
		waitTail(t, tail)
	})

	t.Run("2.2.2.3 fulfill pending more than once, delayed", func(t *testing.T) {
		t.Parallel()
		p, resolve, _ := deferred()
		times := 0
		tail := p.Then(func(v any) (any, error) {
			times++
			if times != 1 {
				t.Errorf("onFulfilled called %d times, want 1", times)
			}
			return nil, nil
		}, nil)
		el.SetTimeout(func() {
			resolve(dummy)
			resolve(dummy)
		}, 50)
		waitTail(t, tail)
	})

	t.Run("2.2.2.3 fulfill pending more than once, immediately then delayed", func(t *testing.T) {
		t.Parallel()
		p, resolve, _ := deferred()
		times := 0
		tail := p.Then(func(v any) (any, error) {
			times++
			if times != 1 {
				t.Errorf("onFulfilled called %d times, want 1", times)
			}
			return nil, nil
		}, nil)
		resolve(dummy)
		el.SetTimeout(func() { resolve(dummy) }, 50)
		waitTail(t, tail)
	})

	t.Run("2.2.2.3 multiple then calls spaced apart", func(t *testing.T) {
		t.Parallel()
		// adjusted（时间型）：改用固定延时结算后读原子计数，避免 wall-clock 竞态。
		p, resolve, _ := deferred()
		var times [3]atomic.Int32
		p.Then(func(v any) (any, error) { record(t, times[:], 0); return nil, nil }, nil)
		el.SetTimeout(func() {
			p.Then(func(v any) (any, error) { record(t, times[:], 1); return nil, nil }, nil)
		}, 50)
		el.SetTimeout(func() {
			p.Then(func(v any) (any, error) { record(t, times[:], 2); return nil, nil }, nil)
		}, 100)
		el.SetTimeout(func() { resolve(dummy) }, 150)

		time.Sleep(300 * time.Millisecond)
		for i := 0; i < 3; i++ {
			if got := times[i].Load(); got != 1 {
				t.Errorf("handler %d called %d times, want 1", i+1, got)
			}
		}
	})

	t.Run("2.2.2.3 then interleaved with fulfillment", func(t *testing.T) {
		t.Parallel()
		p, resolve, _ := deferred()
		var times [2]atomic.Int32
		p.Then(func(v any) (any, error) { record(t, times[:], 0); return nil, nil }, nil)
		resolve(dummy)
		tail := p.Then(func(v any) (any, error) { record(t, times[:], 1); return nil, nil }, nil)
		waitTail(t, tail)
		for i := 0; i < 2; i++ {
			if got := times[i].Load(); got != 1 {
				t.Errorf("handler %d called %d times, want 1", i+1, got)
			}
		}
	})
}

// record 递增第 i 个调用计数；>1 即触发“被重复调用”的错误。
func record(t *testing.T, times []atomic.Int32, i int) {
	if times[i].Add(1) > 1 {
		t.Errorf("handler %d called more than once", i+1)
	}
}
