package promise_test

import (
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestEach 覆盖 EventLoop.Each 全部分支。
func TestEach(t *testing.T) {
	t.Parallel()

	t.Run("all-success", func(t *testing.T) {
		t.Parallel()
		p1 := el.Resolve(1)
		p2 := el.Resolve(2)

		p := el.Each(func(item any, index int, arrLen int) any {
			return item.(int) * 2
		}, p1, p2, 3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		arr := p.Value().([]any)
		if len(arr) != 3 {
			t.Fatalf("Expected array length 3, got %d", len(arr))
		}
		if arr[0] != 1 || arr[1] != 2 || arr[2] != 3 {
			t.Fatalf("Expected [1, 2, 3], got %v", arr)
		}
	})

	t.Run("one-failure", func(t *testing.T) {
		t.Parallel()
		p2 := el.Reject("error")

		p := el.Each(func(item any, index int, arrLen int) any {
			return item.(int) * 2
		}, 1, p2, 3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "UnexpectedError: error" {
			t.Fatalf("Expected error 'UnexpectedError: error', got %s", got)
		}
	})

	t.Run("iterator-returns-promise", func(t *testing.T) {
		t.Parallel()
		p := el.Each(func(item any, index int, arrLen int) any {
			return el.Resolve(item.(int) * 2)
		}, 1, 2, 3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		arr := p.Value().([]any)
		if len(arr) != 3 {
			t.Fatalf("Expected array length 3, got %d", len(arr))
		}
		if arr[0] != 1 || arr[1] != 2 || arr[2] != 3 {
			t.Fatalf("Expected [1, 2, 3], got %v", arr)
		}
	})

	t.Run("empty-array", func(t *testing.T) {
		t.Parallel()
		p := el.Each(func(item any, index int, arrLen int) any {
			return item
		}, make([]any, 0)...)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if arr := p.Value().([]any); len(arr) != 0 {
			t.Fatalf("Expected empty array, got %v", arr)
		}
	})

	t.Run("nil-iterator", func(t *testing.T) {
		t.Parallel()
		p := el.Each(nil, 1, 2)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "TypeError: nil is not a function" {
			t.Fatalf("Expected 'TypeError: nil is not a function', got %s", got)
		}
	})

	t.Run("nil-input", func(t *testing.T) {
		t.Parallel()
		p := el.Each(func(item any, index int, arrLen int) any {
			return item
		})
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "TypeError: nil is not iterable" {
			t.Fatalf("Expected 'TypeError: nil is not iterable', got %s", got)
		}
	})

	t.Run("iterator-panic", func(t *testing.T) {
		t.Parallel()
		p := el.Each(func(item any, index int, arrLen int) any {
			panic("each boom")
		}, el.Resolve(1), el.Resolve(2))
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
	})
}
