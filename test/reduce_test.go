package promise_test

import (
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestReduce 覆盖 EventLoop.Reduce 全部分支。
func TestReduce(t *testing.T) {
	t.Parallel()

	t.Run("empty-array", func(t *testing.T) {
		t.Parallel()
		reducer := func(acc any, item any) any {
			return acc.(int) + item.(int)
		}
		p := el.Reduce(reducer, el.Resolve(3), []any{}...)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if v := p.Value().(int); v != 3 {
			t.Fatalf("Expected value 3, got %d", v)
		}
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		p1 := el.Resolve(1)
		p2 := 2
		p3 := el.Resolve(3)
		reducer := func(acc any, item any) any {
			return acc.(int) + item.(int)
		}
		p := el.Reduce(reducer, 0, p1, p2, p3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if v := p.Value().(int); v != 6 {
			t.Fatalf("Expected value 6, got %d", v)
		}
	})

	t.Run("single-element-nil-initial", func(t *testing.T) {
		t.Parallel()
		reducer := func(acc any, item any) any {
			return acc.(int) + item.(int)
		}
		p := el.Reduce(reducer, nil, el.Resolve(4))
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if v := p.Value().(int); v != 4 {
			t.Fatalf("Expected value 4, got %d", v)
		}
	})

	t.Run("rejected", func(t *testing.T) {
		t.Parallel()
		reducer := func(acc any, item any) any {
			return acc.(int) + item.(int)
		}
		p := el.Reduce(reducer, el.Resolve(3), el.Resolve(4), el.Reject("error"))
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "UnexpectedError: error" {
			t.Fatalf("Expected error 'UnexpectedError: error', got %s", got)
		}
	})

	t.Run("reducer-panic", func(t *testing.T) {
		t.Parallel()
		p := el.Reduce(func(acc any, cur any) any {
			panic("reduce boom")
		}, 0, el.Resolve(1), el.Resolve(2))
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
	})
}
