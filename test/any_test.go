package promise_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestAny 覆盖 EventLoop.Any 全部分支。
func TestAny(t *testing.T) {
	t.Parallel()

	t.Run("fulfilled", func(t *testing.T) {
		t.Parallel()
		p1 := el.NewPromise(func(resolve, reject func(v any)) error {
			reject("error1")
			return nil
		})
		p2 := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve("success")
			return nil
		})
		p3 := el.NewPromise(func(resolve, reject func(v any)) error {
			reject("error2")
			return nil
		})

		p := el.Any(p1, p2, p3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if v := p.Value(); v != "success" {
			t.Fatalf("Expected value 'success', got %v", v)
		}
	})

	t.Run("all-rejected", func(t *testing.T) {
		t.Parallel()
		p1 := el.NewPromise(func(resolve, reject func(v any)) error {
			reject("error1")
			return nil
		})
		p2 := el.NewPromise(func(resolve, reject func(v any)) error {
			reject("error2")
			return nil
		})

		p := el.Any(p1, p2)
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
		if errs[0].Error() != "UnexpectedError: error1" || errs[1].Error() != "UnexpectedError: error2" {
			t.Fatalf("Expected errors ['UnexpectedError: error1','UnexpectedError: error2'], got %v", errs)
		}
	})

	t.Run("nil-input", func(t *testing.T) {
		t.Parallel()
		p := el.Any()
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
		p := el.Any(make([]any, 0)...)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		var agg *AggregateError
		if !errors.As(p.Reason(), &agg) {
			t.Fatalf("Expected *AggregateError, got %T", p.Reason())
		}
		errs := agg.Unwrap()
		if len(errs) != 0 {
			t.Fatalf("Expected 0 errors, got %d", len(errs))
		}
	})
}
