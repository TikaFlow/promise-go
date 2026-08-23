package promise_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// Timeout：base 永不 settle → 在 millis 后以 *TimeoutError 拒绝。
func TestTimeoutWhenNeverSettles(t *testing.T) {
	p, _, _ := el.PromiseWithResolvers()
	tp := el.Timeout(p, 50)

	mustSettle(t, tp, time.Second)
	if tp.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", tp.State())
	}
	var te *TimeoutError
	if !errors.As(tp.Reason(), &te) {
		t.Fatalf("expect *TimeoutError, got %v", tp.Reason())
	}
}

// Timeout：base 已决（值）→ 跟随其值，且不误超时。
func TestTimeoutWhenAlreadySettled(t *testing.T) {
	tp := el.Timeout("ok", 200)
	mustSettle(t, tp, time.Second)
	if tp.State() != Fulfilled {
		t.Fatalf("expect Fulfilled, got %s", tp.State())
	}
	if tp.Value() != "ok" {
		t.Fatalf("unexpected value: %v", tp.Value())
	}
}

// Timeout：base 晚于 deadline 才 settle → 超时拒绝。
func TestTimeoutWhenSettlesLate(t *testing.T) {
	p, resolve, _ := el.PromiseWithResolvers()
	el.SetTimeout(func() { resolve("too late") }, 300)
	tp := el.Timeout(p, 50)

	mustSettle(t, tp, time.Second)
	if tp.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", tp.State())
	}
	var te *TimeoutError
	if !errors.As(tp.Reason(), &te) {
		t.Fatalf("expect *TimeoutError, got %v", tp.Reason())
	}
}

// Timeout：base 拒绝 → 跟随其拒绝理由（而非超时）。
func TestTimeoutWhenBaseRejected(t *testing.T) {
	base := el.Reject(errors.New("base rejected"))
	tp := el.Timeout(base, 200)
	mustSettle(t, tp, time.Second)
	if tp.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", tp.State())
	}
	if tp.Reason().Error() != "base rejected" {
		t.Fatalf("unexpected reason: %v", tp.Reason())
	}
}

// Timeout：非 promise 值 → 立即跟随，不超时。
func TestTimeoutWithPlainValue(t *testing.T) {
	tp := el.Timeout(42, 200)
	mustSettle(t, tp, time.Second)
	if tp.State() != Fulfilled {
		t.Fatalf("expect Fulfilled, got %s", tp.State())
	}
	if tp.Value() != 42 {
		t.Fatalf("unexpected value: %v", tp.Value())
	}
}
