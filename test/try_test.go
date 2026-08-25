package promise_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestTry 覆盖 EventLoop.Try 全部分支。
func TestTry(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		p := el.Try(func(args ...any) (any, error) {
			return "success", nil
		})
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if v := p.Value(); v != "success" {
			t.Fatalf("Expected value 'success', got %v", v)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		p := el.Try(func(args ...any) (any, error) {
			return nil, errors.New("error value")
		})
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "error value" {
			t.Fatalf("Expected error 'error value', got %s", got)
		}
	})

	t.Run("nil-func", func(t *testing.T) {
		t.Parallel()
		p := el.Try(nil)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "TypeError: Promise executor must be a function" {
			t.Fatalf("Expected 'TypeError: Promise executor must be a function', got %s", got)
		}
	})

	t.Run("returns-promise", func(t *testing.T) {
		t.Parallel()
		p := el.Try(func(args ...any) (any, error) {
			return el.Resolve("absorbed"), nil
		})
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if v := p.Value(); v != "absorbed" {
			t.Fatalf("Expected value 'absorbed', got %v", v)
		}
	})

	t.Run("panic", func(t *testing.T) {
		t.Parallel()
		p := el.Try(func(...any) (any, error) {
			panic(errTryPanic)
		})
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if !errors.Is(p.Reason(), errTryPanic) {
			t.Fatalf("unexpected reason: %v", p.Reason())
		}
	})

	t.Run("panic-triggers-executor-hook", func(t *testing.T) {
		t.Parallel()
		el2 := StartEventLoop(1)
		defer el2.Stop()

		var executorCalls int
		el2.OnPanic(ExecutorPanic, func(r any) {
			executorCalls++
		})

		p := el2.Try(func(...any) (any, error) {
			panic("try boom")
		})
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if executorCalls == 0 {
			t.Errorf("Try 的 fn panic 应触发 ExecutorPanic 钩子")
		}
	})
}
