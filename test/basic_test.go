package promise_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 测试Promise执行器错误处理
func TestExecutorError(t *testing.T) {
	t.Parallel()
	errorMsg := "executor error"

	p := el.NewPromise(func(resolve, reject func(v any)) error {
		return errors.New(errorMsg)
	})

	p.Then(func(v any) (any, error) {
		t.Errorf("Promise should not be fulfilled, got value: %v", v)
		return nil, nil
	}, func(val error) (any, error) {
		if val.Error() != errorMsg {
			t.Errorf("Expected error '%s', got '%s'", errorMsg, val.Error())
		}
		return nil, nil
	})
}

// 测试执行器在resolve或reject已调用后报错
func TestExecutorErrorAfterResolved(t *testing.T) {
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

	p.Then(func(v any) (any, error) {
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise should not be rejected, got reason: %v", v)
		return nil, nil
	})

	time.Sleep(2 * time.Second)
}

// 测试nil执行器
func TestExecutorNil(t *testing.T) {
	t.Parallel()
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("Expected panic for nil executor")
		}
		if _, ok := r.(*TypeError); !ok {
			t.Errorf("Expected *TypeError panic, got %T: %v", r, r)
		}
	}()

	el.NewPromise(nil)
}

// 测试 executor panic 触发 ExecutorPanic 钩子，且 Promise 被拒绝。
func TestExecutorPanicHook(t *testing.T) {
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
}

// 测试 Try 的 fn panic 触发 ExecutorPanic 钩子，且 Promise 被拒绝。
func TestTryFnPanicHook(t *testing.T) {
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
}

// 测试Thenable对象处理
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
