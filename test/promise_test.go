package promise_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 测试Catch方法
func TestCatch(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})

	p1.Catch(func(v error) (any, error) {
		return "caught: " + v.Error(), nil
	}).Then(func(v any) (any, error) {
		if v != "caught: UnexpectedError: error" {
			t.Errorf("Expected value 'caught: UnexpectedError: error', got %v", v)
		}
		return nil, nil
	}, nil)
}

// 测试Catch穿透
func TestCatchPassThrough(t *testing.T) {
	t.Parallel()
	p := el.Resolve("success")

	p.Then(func(v any) (any, error) {
		return v.(string) + " passed 1,", nil
	}, nil).Then(func(v any) (any, error) {
		return v.(string) + " passed 2,", nil
	}, func(r error) (any, error) {
		t.Errorf("Promise should not be rejected, got reason: %v", r.Error())
		return nil, nil
	}).Then(func(v any) (any, error) {
		return nil, errors.New(v.(string) + " rejected")
	}, nil).Then(func(v any) (any, error) {
		t.Errorf("Promise should not be fulfilled, got value: %v", v)
		return nil, nil
	}, nil).Catch(func(r error) (any, error) {
		if r.Error() != "success passed 1, passed 2, rejected" {
			t.Errorf("Expected value 'success passed 1, passed 2, rejected', got %v", r.Error())
		}
		return nil, nil
	})
}

// 测试Finally方法抛出错误
func TestFinallyError(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("success")
		return nil
	})

	p1.Finally(func() (any, error) {
		return nil, errors.New("finally error")
	}).Then(func(v any) (any, error) {
		t.Errorf("Expected promise to be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "finally error" {
			t.Errorf("Expected error value 'finally error', got %v", v.Error())
		}
		return nil, nil
	})
}

// 测试Finally方法 - 成功状态
func TestFinallyFulfilled(t *testing.T) {
	t.Parallel()
	finallyCalled := false

	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("success")
		return nil
	})

	p1.Finally(func() (any, error) {
		finallyCalled = true
		return nil, nil
	}).Then(func(v any) (any, error) {
		if !finallyCalled {
			t.Errorf("Finally callback was not called")
		}
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, nil)
}

// 测试Finally方法 - 拒绝状态：原 promise 拒绝时，应沿用原拒绝理由（而非以理由作为解决值）。
func TestFinallyRejected(t *testing.T) {
	t.Parallel()
	finallyCalled := false

	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject(errors.New("error"))
		return nil
	})

	d := p1.Finally(func() (any, error) {
		finallyCalled = true
		return nil, nil
	})
	_, err := el.Await(d, 200)
	if err == nil || err.Error() != "error" {
		t.Fatalf("expected rejection with 'error', got v=%v err=%v", d.Value(), err)
	}
	if !finallyCalled {
		t.Errorf("Finally callback was not called")
	}
}

// 测试Finally方法返回被拒绝的Promise
func TestFinallyReturnsRejectedPromise(t *testing.T) {
	t.Parallel()
	rejectedPromise := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("rejected from finally")
		return nil
	})

	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("success")
		return nil
	})

	p1.Finally(func() (any, error) {
		return rejectedPromise, nil
	}).Then(func(v any) (any, error) {
		t.Errorf("Expected promise to be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "UnexpectedError: rejected from finally" {
			t.Errorf("Expected rejection reason 'UnexpectedError: rejected from finally', got %s", v.Error())
		}
		return nil, nil
	})
}

// onFinally 返回一个已解决的 promise：其值被丢弃，新 promise 仍以原拒绝理由拒绝。
func TestFinallyRejectedDiscardFulfilledPromise(t *testing.T) {
	t.Parallel()
	p1 := el.Reject(errors.New("boom"))
	d := p1.Finally(func() (any, error) {
		return el.Resolve("ignored"), nil
	})
	_, err := el.Await(d, 200)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected rejection with 'boom', got v=%v err=%v", d.Value(), err)
	}
}

// onFinally 返回一个已解决的 promise：其值被丢弃，新 promise 以原解决值解决。
func TestFinallyFulfilledDiscardFulfilledPromise(t *testing.T) {
	t.Parallel()
	p1 := el.Resolve("orig")
	d := p1.Finally(func() (any, error) {
		return el.Resolve("ignored"), nil
	})
	v, err := el.Await(d, 200)
	if err != nil || v != "orig" {
		t.Fatalf("expected fulfillment with 'orig', got v=%v err=%v", v, err)
	}
}

// onFinally 返回未决 promise 时不等待，新 promise 立即沿用原状态；
// 该 promise 稍后拒绝也不影响新 promise（与 MDN 在“等待未决”上的设计取舍）。
func TestFinallyPendingNotAwaited(t *testing.T) {
	t.Parallel()
	p1 := el.Resolve("orig")
	p, _, rejectP := el.PromiseWithResolvers()
	d := p1.Finally(func() (any, error) {
		return p, nil
	})
	v, err := el.Await(d, 200)
	if err != nil || v != "orig" {
		t.Fatalf("expected fulfillment with 'orig' sans wait, got v=%v err=%v", v, err)
	}
	el.SetTimeout(func() { rejectP(errors.New("late")) }, 50)
	time.Sleep(2 * time.Second)
	if d.State() != Fulfilled || d.Value() != "orig" {
		t.Fatalf("expected Fulfilled/'orig' after late reject, got %s/%v", d.State(), d.Value())
	}
}

// 测试String函数的格式化输出
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

// 测试State方法
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

// 测试Value方法
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

// 测试Reason方法
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

// 测试Done方法
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
