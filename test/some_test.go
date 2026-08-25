package promise_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestSome 覆盖 EventLoop.Some 全部分支。
func TestSome(t *testing.T) {
	t.Parallel()

	t.Run("two-of-three", func(t *testing.T) {
		t.Parallel()
		p1 := el.Resolve("success1")
		p2 := el.Resolve("success2")
		p3 := el.Reject("failure")

		p := el.Some(2, p1, p2, p3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		values := p.Value().([]any)
		if len(values) != 2 {
			t.Fatalf("Expected 2 values, got %d", len(values))
		}
		if values[0] != "success1" || values[1] != "success2" {
			t.Fatalf("Expected values ['success1','success2'], got %v", values)
		}
	})

	t.Run("too-many-rejected", func(t *testing.T) {
		t.Parallel()
		p1 := el.Reject("failure1")
		p2 := el.Reject("failure2")
		p3 := el.Resolve("success")

		p := el.Some(2, p1, p2, p3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		var agg *AggregateError
		if !errors.As(p.Reason(), &agg) {
			t.Fatalf("Expected *AggregateError, got %T", p.Reason())
		}
		errs := agg.Unwrap()
		if len(errs) != 2 {
			t.Fatalf("Expected 2 errors, got %d", len(errs))
		}
		if errs[0].Error() != "UnexpectedError: failure1" || errs[1].Error() != "UnexpectedError: failure2" {
			t.Fatalf("Expected errors ['UnexpectedError: failure1','UnexpectedError: failure2'], got %v", errs)
		}
	})

	t.Run("num-greater-than-len", func(t *testing.T) {
		t.Parallel()
		p := el.Some(2, el.Resolve("success"))
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "RangeError: no enough promises to resolve" {
			t.Fatalf("Expected 'RangeError: no enough promises to resolve', got %s", got)
		}
	})

	t.Run("num-le-zero", func(t *testing.T) {
		t.Parallel()
		p := el.Some(0, el.Resolve("success"))
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "RangeError: num must be greater than 0" {
			t.Fatalf("Expected 'RangeError: num must be greater than 0', got %s", got)
		}
	})

	t.Run("nil-input", func(t *testing.T) {
		t.Parallel()
		p := el.Some(2)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "TypeError: nil is not iterable" {
			t.Fatalf("Expected 'TypeError: nil is not iterable', got %s", got)
		}
	})

	t.Run("empty-array", func(t *testing.T) {
		t.Parallel()
		p := el.Some(2, make([]any, 0)...)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		var agg *AggregateError
		if !errors.As(p.Reason(), &agg) {
			t.Fatalf("Expected *AggregateError, got %T", p.Reason())
		}
		if errs := agg.Unwrap(); len(errs) != 0 {
			t.Fatalf("Expected 0 errors, got %d", len(errs))
		}
	})
}
