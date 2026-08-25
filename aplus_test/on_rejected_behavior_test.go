package aplus_test

import (
	"sync/atomic"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 2.2.3: 若 onRejected 是函数，则在 promise rejected 后以其理由为参调用。
func TestAplus2_2_3(t *testing.T) {
	t.Parallel()
	// 2.2.3.1 with rejection reason
	testRejected(t, sentinel, func(t *testing.T, p *Promise) *Promise {
		return expectRejected(t, p, sentinel)
	})

	// 2.2.3.2 not before rejection
	t.Run("2.2.3.2 rejected after a delay", func(t *testing.T) {
		t.Parallel()
		p, _, reject := deferred()
		isRejected := false
		tail := p.Then(nil, func(r error) (any, error) {
			if !isRejected {
				t.Errorf("onRejected called before promise was rejected")
			}
			return nil, nil
		})
		el.SetTimeout(func() {
			isRejected = true
			reject(dummy)
		}, 50)
		waitTail(t, tail)
	})

	t.Run("2.2.3.2 never rejected", func(t *testing.T) {
		t.Parallel()
		p, _, _ := deferred()
		var onRejectedCalled atomic.Bool
		p.Then(nil, func(r error) (any, error) {
			onRejectedCalled.Store(true)
			return nil, nil
		})
		time.Sleep(150 * time.Millisecond)
		if onRejectedCalled.Load() {
			t.Errorf("onRejected must not be called if the promise is never rejected")
		}
	})

	// 2.2.3.3 not more than once
	t.Run("2.2.3.3 already-rejected", func(t *testing.T) {
		t.Parallel()
		times := 0
		tail := rejected(dummy).Then(nil, func(r error) (any, error) {
			times++
			if times != 1 {
				t.Errorf("onRejected called %d times, want 1", times)
			}
			return nil, nil
		})
		waitTail(t, tail)
	})

	t.Run("2.2.3.3 reject pending more than once, immediately", func(t *testing.T) {
		t.Parallel()
		p, _, reject := deferred()
		times := 0
		tail := p.Then(nil, func(r error) (any, error) {
			times++
			if times != 1 {
				t.Errorf("onRejected called %d times, want 1", times)
			}
			return nil, nil
		})
		reject(dummy)
		reject(dummy)
		waitTail(t, tail)
	})

	t.Run("2.2.3.3 reject pending more than once, delayed", func(t *testing.T) {
		t.Parallel()
		p, _, reject := deferred()
		times := 0
		tail := p.Then(nil, func(r error) (any, error) {
			times++
			if times != 1 {
				t.Errorf("onRejected called %d times, want 1", times)
			}
			return nil, nil
		})
		el.SetTimeout(func() {
			reject(dummy)
			reject(dummy)
		}, 50)
		waitTail(t, tail)
	})

	t.Run("2.2.3.3 reject pending more than once, immediately then delayed", func(t *testing.T) {
		t.Parallel()
		p, _, reject := deferred()
		times := 0
		tail := p.Then(nil, func(r error) (any, error) {
			times++
			if times != 1 {
				t.Errorf("onRejected called %d times, want 1", times)
			}
			return nil, nil
		})
		reject(dummy)
		el.SetTimeout(func() { reject(dummy) }, 50)
		waitTail(t, tail)
	})

	t.Run("2.2.3.3 multiple then calls spaced apart", func(t *testing.T) {
		t.Parallel()
		// adjusted（时间型）：固定延时结算后读原子计数。
		p, _, reject := deferred()
		var times [3]atomic.Int32
		p.Then(nil, func(r error) (any, error) { record(t, times[:], 0); return nil, nil })
		el.SetTimeout(func() {
			p.Then(nil, func(r error) (any, error) { record(t, times[:], 1); return nil, nil })
		}, 50)
		el.SetTimeout(func() {
			p.Then(nil, func(r error) (any, error) { record(t, times[:], 2); return nil, nil })
		}, 100)
		el.SetTimeout(func() { reject(dummy) }, 150)

		time.Sleep(300 * time.Millisecond)
		for i := 0; i < 3; i++ {
			if got := times[i].Load(); got != 1 {
				t.Errorf("handler %d called %d times, want 1", i+1, got)
			}
		}
	})

	t.Run("2.2.3.3 then interleaved with rejection", func(t *testing.T) {
		t.Parallel()
		p, _, reject := deferred()
		var times [2]atomic.Int32
		p.Then(nil, func(r error) (any, error) { record(t, times[:], 0); return nil, nil })
		reject(dummy)
		tail := p.Then(nil, func(r error) (any, error) { record(t, times[:], 1); return nil, nil })
		waitTail(t, tail)
		for i := 0; i < 2; i++ {
			if got := times[i].Load(); got != 1 {
				t.Errorf("handler %d called %d times, want 1", i+1, got)
			}
		}
	})
}
