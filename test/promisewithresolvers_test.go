package promise_test

import (
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestPromiseWithResolvers 覆盖 EventLoop.PromiseWithResolvers 的 resolve/reject 分支。
func TestPromiseWithResolvers(t *testing.T) {
	t.Parallel()

	t.Run("resolve", func(t *testing.T) {
		t.Parallel()
		p, resolve, _ := el.PromiseWithResolvers()
		resolve("resolved value")

		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if v := p.Value(); v != "resolved value" {
			t.Fatalf("Expected value 'resolved value', got %v", v)
		}
	})

	t.Run("reject", func(t *testing.T) {
		t.Parallel()
		p, _, reject := el.PromiseWithResolvers()
		reject("rejected reason")

		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "UnexpectedError: rejected reason" {
			t.Fatalf("Expected 'UnexpectedError: rejected reason', got %s", got)
		}
	})
}
