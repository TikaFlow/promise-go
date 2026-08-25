package promise_test

import (
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestAll 覆盖 EventLoop.All 全部分支。
func TestAll(t *testing.T) {
	t.Parallel()

	t.Run("fulfilled", func(t *testing.T) {
		t.Parallel()
		p1 := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve(1)
			return nil
		})
		p2 := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve(2)
			return nil
		})
		p3 := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve(3)
			return nil
		})

		p := el.All(p1, p2, p3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		results, ok := p.Value().([]any)
		if !ok {
			t.Fatalf("Expected []any type, got %T", p.Value())
		}
		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}
		if results[0] != 1 || results[1] != 2 || results[2] != 3 {
			t.Fatalf("Expected [1, 2, 3], got %v", results)
		}
	})

	t.Run("nil-input", func(t *testing.T) {
		t.Parallel()
		p := el.All()
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
		p := el.All(make([]any, 0)...)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		results, ok := p.Value().([]any)
		if !ok {
			t.Fatalf("Expected []any type, got %T", p.Value())
		}
		if len(results) != 0 {
			t.Fatalf("Expected empty array, got %d elements", len(results))
		}
	})

	t.Run("not-promise", func(t *testing.T) {
		t.Parallel()
		p1 := "string"
		p2 := 2
		p3 := false

		p := el.All(p1, p2, p3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		results, ok := p.Value().([]any)
		if !ok {
			t.Fatalf("Expected []any type, got %T", p.Value())
		}
		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}
		if results[0] != p1 || results[1] != p2 || results[2] != p3 {
			t.Fatalf("Expected [%v, %v, %v], got %v", p1, p2, p3, results)
		}
	})

	t.Run("rejected", func(t *testing.T) {
		t.Parallel()
		p1 := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve(1)
			return nil
		})
		p2 := el.NewPromise(func(resolve, reject func(v any)) error {
			reject("error")
			return nil
		})
		p3 := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve(3)
			return nil
		})

		p := el.All(p1, p2, p3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "UnexpectedError: error" {
			t.Fatalf("Expected rejection 'UnexpectedError: error', got %s", got)
		}
	})
}
