package promise_test

import (
	"errors"
	"testing"
	"time"
)

// 测试Await - 拒绝的Promise
func TestAwaitRejectedPromise(t *testing.T) {
	t.Parallel()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})

	_, err := el.Await(p, 50)
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else if err.Error() != "UnexpectedError: error" {
		t.Errorf("Expected error 'UnexpectedError: error', got %s", err)
	}
}

// 测试Await - 成功
func TestAwaitSuccess(t *testing.T) {
	t.Parallel()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("success")
		return nil
	})

	res, err := el.Await(p, 50)
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if res != "success" {
		t.Errorf("Expected 'success', got %s", res)
	}
}

// 测试Await - 超时
func TestAwaitTimeout(t *testing.T) {
	t.Parallel()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		el.SetTimeout(func() {
			resolve("success")
		}, 100)
		return nil
	})

	_, err := el.Await(p, 50)
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else if err.Error() != "TimeoutError: await timeout" {
		t.Errorf("Expected error 'TimeoutError: await timeout', got %s", err)
	}

	time.Sleep(time.Second)
}

// 测试Await - timeout不是正数
func TestAwaitTimeoutNotPositive(t *testing.T) {
	t.Parallel()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("success")
		return nil
	})

	expected := "RangeError: await timeout must be greater than 0"
	_, err := el.Await(p, -100)
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else if err.Error() != expected {
		t.Errorf("Expected error '%s', got %s", expected, err)
	}
}

// 测试Delay函数
func TestDelay(t *testing.T) {
	t.Parallel()
	val := "value"

	if _, err := el.Await(el.Delay(val, 50), 40); err == nil {
		// timeout
		t.Errorf("Expected 'TimeoutError: await timeout', got nil")
	}

	res, err := el.Await(el.Delay(val, 50), 100)
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if res != val {
		t.Errorf("Expected '%s', got %s", val, res)
	}

	p := el.NewPromise(func(resolve, reject func(v any)) error {
		el.SetTimeout(func() {
			resolve("success")
		}, 50)
		return nil
	})
	if _, err = el.Await(el.Delay(p, 50), 90); err == nil {
		// timeout
		t.Errorf("Expected 'TimeoutError: await timeout', got nil")
	}

	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		el.SetTimeout(func() {
			resolve("success")
		}, 50)
		return nil
	})
	if res, _ := el.Await(el.Delay(p2, 50), 150); res != "success" {
		t.Errorf("Expected 'success', got %s", res)
	}

	time.Sleep(time.Second)
}

// 测试Delay函数 - 已拒绝的promise
func TestDelayRejectedPromise(t *testing.T) {
	t.Parallel()
	rejected := el.Reject("rejected reason")
	p := el.Delay(rejected, 50)

	_, err := el.Await(p, 100)
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else if err.Error() != "UnexpectedError: rejected reason" {
		t.Errorf("Expected 'UnexpectedError: rejected reason', got %v", err)
	}
}

// 测试PromiseWithResolvers方法
func TestPromiseWithResolvers(t *testing.T) {
	t.Parallel()
	// 测试成功情况
	p, resolve, _ := el.PromiseWithResolvers()
	resolve("resolved value")

	p.Then(func(v any) (any, error) {
		if v != "resolved value" {
			t.Errorf("Expected value 'resolved value', got %v", v)
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise should be fulfilled, but was rejected with %v", v.Error())
		return nil, nil
	})

	// 测试拒绝情况
	p2, _, reject2 := el.PromiseWithResolvers()
	reject2("rejected reason")

	p2.Then(func(v any) (any, error) {
		t.Errorf("Promise should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "UnexpectedError: rejected reason" {
			t.Errorf("Expected value 'UnexpectedError: rejected reason', got '%s'", v.Error())
		}
		return nil, nil
	})
}

// 测试Try方法 - 失败
func TestTryError(t *testing.T) {
	t.Parallel()
	el.Try(func(args ...any) (any, error) {
		return nil, errors.New("error value")
	}).Then(func(v any) (any, error) {
		t.Errorf("Promise.Try should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "error value" {
			t.Errorf("Expected error value 'error value', got %v", v.Error())
		}
		return nil, nil
	})
}

// 测试Try方法 - nil函数
func TestTryNilFunc(t *testing.T) {
	t.Parallel()
	el.Try(nil).Then(func(v any) (any, error) {
		t.Errorf("Promise.Try with nil func should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(val error) (any, error) {
		if val.Error() != "TypeError: Promise executor must be a function" {
			t.Errorf("Expected error message, got %s", val.Error())
		}
		return nil, nil
	})
}

// 测试Try方法 - 成功
func TestTrySuccess(t *testing.T) {
	t.Parallel()
	el.Try(func(args ...any) (any, error) {
		return "success", nil
	}).Then(func(v any) (any, error) {
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise.Try should be fulfilled, but was rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试Async方法 - 正常返回
func TestAsyncSuccess(t *testing.T) {
	t.Parallel()
	p := el.Async(func() (any, error) {
		time.Sleep(time.Millisecond * 50)
		return "async result", nil
	})

	res, err := el.Await(p, 100)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if res != "async result" {
		t.Errorf("Expected 'async result', got %v", res)
	}
}

// 测试Async方法 - 报错
func TestAsyncError(t *testing.T) {
	t.Parallel()
	p := el.Async(func() (any, error) {
		time.Sleep(time.Millisecond * 50)
		return nil, errors.New("async error")
	})

	_, err := el.Await(p, 100)
	if err == nil {
		t.Errorf("Expected error, got nil")
	} else if err.Error() != "async error" {
		t.Errorf("Expected 'async error', got %v", err)
	}
}

// 测试Async方法 - 返回promise
func TestAsyncReturnsPromise(t *testing.T) {
	t.Parallel()
	p := el.Async(func() (any, error) {
		return el.NewPromise(func(resolve, reject func(v any)) error {
			resolve("async promise")
			return nil
		}), nil
	})

	res, err := el.Await(p, 100)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if res != "async promise" {
		t.Errorf("Expected 'async promise', got %v", res)
	}
}
