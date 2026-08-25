package promise_test

import (
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestResolve 覆盖 EventLoop.Resolve 全部分支。
func TestResolve(t *testing.T) {
	t.Parallel()

	t.Run("plain-value", func(t *testing.T) {
		t.Parallel()
		p := el.Resolve("value")
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if v := p.Value(); v != "value" {
			t.Fatalf("Expected value 'value', got %v", v)
		}
	})

	t.Run("fulfilled-promise", func(t *testing.T) {
		t.Parallel()
		original := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve("original")
			return nil
		})

		p := el.Resolve(original)
		if p != original {
			t.Fatalf("Expected Resolve to return the same Promise instance")
		}
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if v := p.Value(); v != "original" {
			t.Fatalf("Expected value 'original', got %v", v)
		}
	})

	t.Run("rejected-promise", func(t *testing.T) {
		t.Parallel()
		original := el.NewPromise(func(resolve, reject func(v any)) error {
			reject("error")
			return nil
		})

		p := el.Resolve(original)
		if p != original {
			t.Fatalf("Expected Resolve to return the same Promise instance")
		}
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "UnexpectedError: error" {
			t.Fatalf("Expected 'UnexpectedError: error', got %s", got)
		}
	})
}

// TestReject 覆盖 EventLoop.Reject 分支。
func TestReject(t *testing.T) {
	t.Parallel()
	p := el.Reject("reason")
	mustSettle(t, p, 2*time.Second)
	if p.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", p.State())
	}
	if got := p.Reason().Error(); got != "UnexpectedError: reason" {
		t.Fatalf("Expected 'UnexpectedError: reason', got %s", got)
	}
}
