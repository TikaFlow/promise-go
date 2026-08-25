package promise_test

import (
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestMap 覆盖 EventLoop.Map 全部分支。
func TestMap(t *testing.T) {
	t.Parallel()

	t.Run("all-resolved", func(t *testing.T) {
		t.Parallel()
		p1 := el.Resolve(1)
		p2 := el.Resolve(2)
		p3 := 3
		mapper := func(item any) any {
			return item.(int) * 2
		}

		p := el.Map(mapper, p1, p2, p3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		arr := p.Value().([]any)
		if len(arr) != 3 {
			t.Fatalf("Expected array length 3, got %d", len(arr))
		}
		if arr[0] != 2 || arr[1] != 4 || arr[2] != 6 {
			t.Fatalf("Expected [2, 4, 6], got %v", arr)
		}
	})

	t.Run("one-rejected", func(t *testing.T) {
		t.Parallel()
		p1 := el.Resolve(1)
		p2 := el.Reject("error")
		p3 := 3
		mapper := func(item any) any {
			return item.(int) * 2
		}

		p := el.Map(mapper, p1, p2, p3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "UnexpectedError: error" {
			t.Fatalf("Expected error 'UnexpectedError: error', got %s", got)
		}
	})

	t.Run("nil-input", func(t *testing.T) {
		t.Parallel()
		p := el.Map(func(item any) any { return item })
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "TypeError: nil is not iterable" {
			t.Fatalf("Expected 'TypeError: nil is not iterable', got %s", got)
		}
	})

	t.Run("nil-mapper", func(t *testing.T) {
		t.Parallel()
		p := el.Map(nil, 1, 2)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "TypeError: nil is not a function" {
			t.Fatalf("Expected 'TypeError: nil is not a function', got %s", got)
		}
	})

	t.Run("mapper-panic", func(t *testing.T) {
		t.Parallel()
		p := el.Map(func(item any) any {
			panic("map boom")
		}, el.Resolve(1), el.Resolve(2))
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
	})
}
