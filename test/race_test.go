package promise_test

import (
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestRace 覆盖 EventLoop.Race 全部分支。
func TestRace(t *testing.T) {
	t.Parallel()

	t.Run("fulfilled", func(t *testing.T) {
		t.Parallel()
		p1 := el.NewPromise(func(resolve, reject func(v any)) error {
			el.SetTimeout(func() {
				reject("error")
			}, 100)
			return nil
		})
		p2 := el.NewPromise(func(resolve, reject func(v any)) error {
			el.SetTimeout(func() {
				resolve("success")
			}, 50)
			return nil
		})

		p := el.Race(p1, p2)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if v := p.Value(); v != "success" {
			t.Fatalf("Expected value 'success', got %v", v)
		}
	})

	t.Run("rejected", func(t *testing.T) {
		t.Parallel()
		p1 := el.NewPromise(func(resolve, reject func(v any)) error {
			el.SetTimeout(func() {
				reject("error")
			}, 50)
			return nil
		})
		p2 := el.NewPromise(func(resolve, reject func(v any)) error {
			el.SetTimeout(func() {
				resolve("success")
			}, 100)
			return nil
		})

		p := el.Race(p1, p2)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "UnexpectedError: error" {
			t.Fatalf("Expected 'UnexpectedError: error', got %s", got)
		}
	})

	t.Run("nil-input", func(t *testing.T) {
		t.Parallel()
		p := el.Race()
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "TypeError: nil is not iterable" {
			t.Fatalf("Expected error 'TypeError: nil is not iterable', got %s", got)
		}
	})

	t.Run("empty-array", func(t *testing.T) {
		t.Parallel()
		p := el.Race(make([]any, 0)...)
		_, err := el.Await(p, 100)
		if err == nil || err.Error() != "TimeoutError: await timeout" {
			t.Fatalf("Expected 'TimeoutError: await timeout', got %v", err)
		}
		if p.State() != Pending {
			t.Fatalf("Expected state Pending for empty array Race, got %s", p.State())
		}
	})
}
