package promise_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestAsync 覆盖 EventLoop.Async 全部分支。
func TestAsync(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		p := el.Async(func() (any, error) {
			time.Sleep(time.Millisecond * 50)
			return "async result", nil
		})
		v, err := el.Await(p, 100)
		if err != nil {
			t.Fatalf("Expected nil error, got %v", err)
		}
		if v != "async result" {
			t.Fatalf("Expected 'async result', got %v", v)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		p := el.Async(func() (any, error) {
			time.Sleep(time.Millisecond * 50)
			return nil, errors.New("async error")
		})
		_, err := el.Await(p, 100)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}
		if err.Error() != "async error" {
			t.Fatalf("Expected 'async error', got %v", err)
		}
	})

	t.Run("returns-promise", func(t *testing.T) {
		t.Parallel()
		p := el.Async(func() (any, error) {
			return el.NewPromise(func(resolve, reject func(v any)) error {
				resolve("async promise")
				return nil
			}), nil
		})
		v, err := el.Await(p, 100)
		if err != nil {
			t.Fatalf("Expected nil error, got %v", err)
		}
		if v != "async promise" {
			t.Fatalf("Expected 'async promise', got %v", v)
		}
	})

	t.Run("nil-fn", func(t *testing.T) {
		t.Parallel()
		p := el.Async(nil)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "TypeError: fn must be a function" {
			t.Fatalf("Expected 'TypeError: fn must be a function', got %s", got)
		}
	})

	t.Run("panic", func(t *testing.T) {
		t.Parallel()
		p := el.Async(func() (any, error) {
			panic(errAsyncPanic)
		})
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if !errors.Is(p.Reason(), errAsyncPanic) {
			t.Fatalf("unexpected reason: %v", p.Reason())
		}
	})
}
