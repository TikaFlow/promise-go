package promise_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestTimeout 覆盖 EventLoop.Timeout 全部分支。
func TestTimeout(t *testing.T) {
	t.Parallel()

	t.Run("never-settles", func(t *testing.T) {
		t.Parallel()
		p, _, _ := el.PromiseWithResolvers()
		tp := el.Timeout(p, 50)

		mustSettle(t, tp, 2*time.Second)
		if tp.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", tp.State())
		}
		var te *TimeoutError
		if !errors.As(tp.Reason(), &te) {
			t.Fatalf("expect *TimeoutError, got %v", tp.Reason())
		}
	})

	t.Run("already-settled", func(t *testing.T) {
		t.Parallel()
		tp := el.Timeout("ok", 200)
		mustSettle(t, tp, 2*time.Second)
		if tp.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", tp.State())
		}
		if v := tp.Value(); v != "ok" {
			t.Fatalf("unexpected value: %v", v)
		}
	})

	t.Run("settles-late", func(t *testing.T) {
		t.Parallel()
		p, resolve, _ := el.PromiseWithResolvers()
		el.SetTimeout(func() { resolve("too late") }, 300)
		tp := el.Timeout(p, 50)

		mustSettle(t, tp, 2*time.Second)
		if tp.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", tp.State())
		}
		var te *TimeoutError
		if !errors.As(tp.Reason(), &te) {
			t.Fatalf("expect *TimeoutError, got %v", tp.Reason())
		}
	})

	t.Run("base-rejected", func(t *testing.T) {
		t.Parallel()
		base := el.Reject(errors.New("base rejected"))
		tp := el.Timeout(base, 200)
		mustSettle(t, tp, 2*time.Second)
		if tp.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", tp.State())
		}
		if got := tp.Reason().Error(); got != "base rejected" {
			t.Fatalf("unexpected reason: %v", got)
		}
	})

	t.Run("plain-value", func(t *testing.T) {
		t.Parallel()
		tp := el.Timeout(42, 200)
		mustSettle(t, tp, 2*time.Second)
		if tp.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", tp.State())
		}
		if v := tp.Value(); v != 42 {
			t.Fatalf("unexpected value: %v", v)
		}
	})

	t.Run("negative-millis-treats-as-zero", func(t *testing.T) {
		t.Parallel()
		// millis 负值按 0 处理：base 永不 settle → 立即超时拒绝。
		p, _, _ := el.PromiseWithResolvers()
		tp := el.Timeout(p, -1)
		mustSettle(t, tp, 2*time.Second)
		if tp.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", tp.State())
		}
		var te *TimeoutError
		if !errors.As(tp.Reason(), &te) {
			t.Fatalf("expect *TimeoutError, got %v", tp.Reason())
		}
	})
}
