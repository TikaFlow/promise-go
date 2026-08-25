package promise_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestState 覆盖 Promise.State 三个状态分支。
func TestState(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		return nil
	})
	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("value")
		return nil
	})
	p3 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})

	time.Sleep(2 * time.Second)
	if p1.State() != Pending {
		t.Errorf("Expected state Pending, got %s", p1.State())
	}
	if p2.State() != Fulfilled {
		t.Errorf("Expected state Fulfilled, got %s", p2.State())
	}
	if p3.State() != Rejected {
		t.Errorf("Expected state Rejected, got %s", p3.State())
	}
}

// TestValue 覆盖 Promise.Value 各状态下的取值分支。
func TestValue(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		return nil
	})
	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("success")
		return nil
	})
	p3 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})

	if p1.Value() != nil {
		t.Errorf("Expected Value() to be nil for Pending promise, got %v", p1.Value())
	}
	if p2.Value() != "success" {
		t.Errorf("Expected Value() to be 'success', got %v", p2.Value())
	}
	if p3.Value() != nil {
		t.Errorf("Expected Value() to be nil for Rejected promise, got %v", p3.Value())
	}
}

// TestReason 覆盖 Promise.Reason 各状态下的取值分支。
func TestReason(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		return nil
	})
	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("success")
		return nil
	})
	p3 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})

	if p1.Reason() != nil {
		t.Errorf("Expected Reason() to be nil for Pending promise, got %v", p1.Reason())
	}
	if p2.Reason() != nil {
		t.Errorf("Expected Reason() to be nil for Fulfilled promise, got %v", p2.Reason())
	}
	if p3.Reason() == nil || p3.Reason().Error() != "UnexpectedError: error" {
		t.Errorf("Expected Reason() to be 'UnexpectedError: error', got %v", p3.Reason())
	}
}

// TestDone 覆盖 Promise.Done 通道关闭行为。
func TestDone(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		return nil
	})
	select {
	case <-p1.Done():
		t.Errorf("Expected Done() to block for Pending promise")
	default:
	}

	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("success")
		return nil
	})
	<-p2.Done()

	p3 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})
	<-p3.Done()
}

// TestString 覆盖 Promise.String 的格式化输出。
func TestString(t *testing.T) {
	t.Parallel()
	el.SetTimeout(func() {
		p := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve("success")
			return nil
		})

		expected := "Promise<fulfilled>, value: success"
		result := fmt.Sprintf("%s", p)
		if result != expected {
			t.Errorf("Expected string '%s', got '%s'", expected, result)
		}
	}, 0)

	time.Sleep(2 * time.Second)
}

// TestOnFulfilledPanic 覆盖 2.2.7.2：onFulfilled 内 panic 应作为拒绝理由。
func TestOnFulfilledPanic(t *testing.T) {
	t.Parallel()
	p := el.Resolve("value")
	p2 := p.Then(func(v any) (any, error) {
		panic(errOnFulfilledPanic)
	}, nil)

	mustSettle(t, p2, 2*time.Second)
	if p2.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", p2.State())
	}
	if !errors.Is(p2.Reason(), errOnFulfilledPanic) {
		t.Fatalf("unexpected reason: %v", p2.Reason())
	}
}

// TestOnRejectedPanic 覆盖 2.2.7.2 镜像：onRejected 内 panic 应作为拒绝理由。
func TestOnRejectedPanic(t *testing.T) {
	t.Parallel()
	p := el.Reject(errors.New("root reason"))
	p2 := p.Then(nil, func(r error) (any, error) {
		panic(errOnRejectedPanic)
	})

	mustSettle(t, p2, 2*time.Second)
	if p2.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", p2.State())
	}
	if !errors.Is(p2.Reason(), errOnRejectedPanic) {
		t.Fatalf("unexpected reason: %v", p2.Reason())
	}
}

// TestOnFulfilledPanicWithNonError 覆盖：非 error 的 panic 值应被包装为 *UnexpectedError。
func TestOnFulfilledPanicWithNonError(t *testing.T) {
	t.Parallel()
	p := el.Resolve("value")
	p2 := p.Then(func(v any) (any, error) {
		panic("a plain string")
	}, nil)

	mustSettle(t, p2, 2*time.Second)
	if p2.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", p2.State())
	}
	if _, ok := p2.Reason().(*UnexpectedError); !ok {
		t.Fatalf("expect *UnexpectedError, got %T", p2.Reason())
	}
}
