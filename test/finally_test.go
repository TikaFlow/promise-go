package promise_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestFinally 覆盖 Promise.Finally 全部分支。
func TestFinally(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		p1 := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve("success")
			return nil
		})
		d := p1.Finally(func() (any, error) {
			return nil, errors.New("finally error")
		})
		mustSettle(t, d, 2*time.Second)
		if d.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", d.State())
		}
		if got := d.Reason().Error(); got != "finally error" {
			t.Fatalf("Expected error 'finally error', got %s", got)
		}
	})

	t.Run("fulfilled", func(t *testing.T) {
		t.Parallel()
		finallyCalled := false
		p1 := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve("success")
			return nil
		})
		d := p1.Finally(func() (any, error) {
			finallyCalled = true
			return nil, nil
		})
		mustSettle(t, d, 2*time.Second)
		if !finallyCalled {
			t.Errorf("Finally callback was not called")
		}
		if d.State() != Fulfilled || d.Value() != "success" {
			t.Fatalf("expect Fulfilled/'success', got %s/%v", d.State(), d.Value())
		}
	})

	t.Run("rejected", func(t *testing.T) {
		t.Parallel()
		finallyCalled := false
		p1 := el.NewPromise(func(resolve, reject func(v any)) error {
			reject(errors.New("error"))
			return nil
		})
		d := p1.Finally(func() (any, error) {
			finallyCalled = true
			return nil, nil
		})
		_, err := el.Await(d, 200)
		if err == nil || err.Error() != "error" {
			t.Fatalf("expected rejection with 'error', got v=%v err=%v", d.Value(), err)
		}
		if !finallyCalled {
			t.Errorf("Finally callback was not called")
		}
	})

	t.Run("returns-rejected-promise", func(t *testing.T) {
		t.Parallel()
		rejectedPromise := el.NewPromise(func(resolve, reject func(v any)) error {
			reject("rejected from finally")
			return nil
		})
		p1 := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve("success")
			return nil
		})
		d := p1.Finally(func() (any, error) {
			return rejectedPromise, nil
		})
		mustSettle(t, d, 2*time.Second)
		if d.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", d.State())
		}
		if got := d.Reason().Error(); got != "UnexpectedError: rejected from finally" {
			t.Fatalf("Expected 'UnexpectedError: rejected from finally', got %s", got)
		}
	})

	t.Run("rejected-discards-fulfilled-promise", func(t *testing.T) {
		t.Parallel()
		p1 := el.Reject(errors.New("boom"))
		d := p1.Finally(func() (any, error) {
			return el.Resolve("ignored"), nil
		})
		_, err := el.Await(d, 200)
		if err == nil || err.Error() != "boom" {
			t.Fatalf("expected rejection with 'boom', got v=%v err=%v", d.Value(), err)
		}
	})

	t.Run("fulfilled-discards-fulfilled-promise", func(t *testing.T) {
		t.Parallel()
		p1 := el.Resolve("orig")
		d := p1.Finally(func() (any, error) {
			return el.Resolve("ignored"), nil
		})
		v, err := el.Await(d, 200)
		if err != nil || v != "orig" {
			t.Fatalf("expected fulfillment with 'orig', got v=%v err=%v", v, err)
		}
	})

	t.Run("pending-not-awaited", func(t *testing.T) {
		t.Parallel()
		p1 := el.Resolve("orig")
		p, _, rejectP := el.PromiseWithResolvers()
		d := p1.Finally(func() (any, error) {
			return p, nil
		})
		v, err := el.Await(d, 200)
		if err != nil || v != "orig" {
			t.Fatalf("expected fulfillment with 'orig' sans wait, got v=%v err=%v", v, err)
		}
		el.SetTimeout(func() { rejectP(errors.New("late")) }, 50)
		time.Sleep(2 * time.Second)
		if d.State() != Fulfilled || d.Value() != "orig" {
			t.Fatalf("expected Fulfilled/'orig' after late reject, got %s/%v", d.State(), d.Value())
		}
	})
}
