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

// 测试Finally方法 - 拒绝状态
func TestFinallyRejected(t *testing.T) {
	t.Parallel()
	finallyCalled := false

	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})

	p1.Finally(func() (any, error) {
		finallyCalled = true
		return nil, nil
	}).Then(nil, func(v error) (any, error) {
		if !finallyCalled {
			t.Errorf("Finally callback was not called")
		}
		if v.Error() != "error" {
			t.Errorf("Expected value 'error', got %v", v.Error())
		}
		return nil, nil
	})
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

	time.Sleep(time.Second)
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

	time.Sleep(time.Second)
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
