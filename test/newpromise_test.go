package promise_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestNewPromise 覆盖 EventLoop.NewPromise 的 executor 全部分支。
func TestNewPromise(t *testing.T) {
	t.Parallel()

	t.Run("executor-error", func(t *testing.T) {
		t.Parallel()
		errorMsg := "executor error"
		p := el.NewPromise(func(resolve, reject func(v any)) error {
			return errors.New(errorMsg)
		})
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != errorMsg {
			t.Fatalf("Expected error '%s', got '%s'", errorMsg, got)
		}
	})

	t.Run("executor-error-after-resolved", func(t *testing.T) {
		t.Parallel()
		delayResolve := el.NewPromise(func(resolve, reject func(v any)) error {
			el.SetTimeout(func() {
				resolve("success")
			}, 100)
			return nil
		})

		p := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve(delayResolve)
			return errors.New("executor error after resolve")
		})
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if v := p.Value(); v != "success" {
			t.Fatalf("Expected value 'success', got %v", v)
		}
	})

	t.Run("executor-nil", func(t *testing.T) {
		t.Parallel()
		defer func() {
			r := recover()
			if r == nil {
				t.Fatalf("Expected panic for nil executor")
			}
			if _, ok := r.(*TypeError); !ok {
				t.Fatalf("Expected *TypeError panic, got %T: %v", r, r)
			}
		}()
		el.NewPromise(nil)
	})

	t.Run("executor-panic", func(t *testing.T) {
		t.Parallel()
		p := el.NewPromise(func(resolve, reject func(v any)) error {
			panic(errExecutorPanic)
		})
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if !errors.Is(p.Reason(), errExecutorPanic) {
			t.Fatalf("unexpected reason: %v", p.Reason())
		}
	})

	t.Run("executor-panic-after-resolve", func(t *testing.T) {
		t.Parallel()
		p := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve("ok")
			panic("after resolve")
		})
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		if v := p.Value(); v != "ok" {
			t.Fatalf("unexpected value: %v", v)
		}
	})

	t.Run("executor-panic-hook", func(t *testing.T) {
		t.Parallel()
		el2 := StartEventLoop(1)
		defer el2.Stop()

		var executorCalls int
		el2.OnPanic(ExecutorPanic, func(r any) {
			executorCalls++
		})

		sentinel := errors.New("exec boom")
		p := el2.NewPromise(func(resolve, reject func(v any)) error {
			panic(sentinel)
		})
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if !errors.Is(p.Reason(), sentinel) {
			t.Fatalf("unexpected reason: %v", p.Reason())
		}
		if executorCalls == 0 {
			t.Errorf("ExecutorPanic 钩子未被触发")
		}
	})
}

// TestExecutorThenable 覆盖 NewPromise 的 executor 以 Promise 解决时的行为与微任务顺序。
// 属跨函数时序验证，保留为独立测试函数（不并入 TestNewPromise 子测试）。
func TestExecutorThenable(t *testing.T) {
	t.Parallel()
	result := "init"

	el.SetTimeout(func() {
		p1, resolveP1, _ := el.PromiseWithResolvers()
		p2 := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve(p1)
			return nil
		})
		resolveP1("thenable value")

		p1.Then(func(v any) (any, error) {
			result += " =>p1-resolved"
			el.QueueMicrotask(func() {
				result += " =>p1:microtask"
			})
			return nil, nil
		}, nil)

		p2.Then(func(v any) (any, error) {
			result += " =>p2<" + v.(string) + ">"
			el.QueueMicrotask(func() {
				result += " =>p2:microtask"
			})
			return nil, nil
		}, nil)

		el.QueueMicrotask(func() {
			result += " =>microtask"
		})
	}, 0)

	el.SetTimeout(func() {
		expected := "init =>p1-resolved =>microtask =>p1:microtask =>p2<thenable value> =>p2:microtask"
		if result != expected {
			t.Errorf("Expected result '\n%s', got '\n%s'", expected, result)
		}
	}, 10)

	time.Sleep(2 * time.Second)
}
