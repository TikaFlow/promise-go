package promise_test

import (
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestFilter 覆盖 EventLoop.Filter 全部分支。
func TestFilter(t *testing.T) {
	t.Parallel()

	t.Run("all-resolved", func(t *testing.T) {
		t.Parallel()
		p1 := 1
		p2 := el.Resolve(2)
		p3 := 3
		filter := func(item any) bool {
			return item.(int) > 1
		}

		p := el.Filter(filter, p1, p2, p3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		arr := p.Value().([]any)
		if len(arr) != 2 {
			t.Fatalf("Expected array length 2, got %d", len(arr))
		}
		if arr[0] != 2 || arr[1] != 3 {
			t.Fatalf("Expected [2, 3], got %v", arr)
		}
	})

	t.Run("one-rejected", func(t *testing.T) {
		t.Parallel()
		p1 := el.Resolve(1)
		p2 := el.Reject("error")
		p3 := 3
		filter := func(item any) bool {
			return item.(int) > 1
		}

		p := el.Filter(filter, p1, p2, p3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "UnexpectedError: error" {
			t.Fatalf("Expected error 'UnexpectedError: error', got %s", got)
		}
	})

	t.Run("empty-array", func(t *testing.T) {
		t.Parallel()
		p := el.Filter(func(item any) bool { return true }, make([]any, 0)...)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if arr := p.Value().([]any); len(arr) != 0 {
			t.Fatalf("Expected empty array, got %v", arr)
		}
	})

	t.Run("nil-input", func(t *testing.T) {
		t.Parallel()
		p := el.Filter(func(item any) bool { return true })
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "TypeError: nil is not iterable" {
			t.Fatalf("Expected 'TypeError: nil is not iterable', got %s", got)
		}
	})

	t.Run("filter-panic", func(t *testing.T) {
		t.Parallel()
		p := el.Filter(func(item any) bool {
			panic("filter boom")
		}, el.Resolve(1), el.Resolve(2))
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
	})
}
