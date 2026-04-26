package promise_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 测试Promise的基本创建和拒绝
func TestBasicReject(t *testing.T) {
	t.Parallel()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("failure")
		return nil
	})

	p.Then(func(v any) (any, error) {
		t.Errorf("Promise should not be fulfilled, got value: %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "UnexpectedError: failure" {
			t.Errorf("Expected rejection reason 'UnexpectedError: failure', got '%s'", v.Error())
		}
		return nil, nil
	})
}

// 测试Promise的基本创建和解决
func TestBasicResolve(t *testing.T) {
	t.Parallel()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("success")
		return nil
	})

	p.Then(func(v any) (any, error) {
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise should not be rejected, got reason: %v", v.Error())
		return nil, nil
	})
}

// 测试循环引用检测
func TestCycleDetection(t *testing.T) {
	t.Parallel()
	initial := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("initial")
		return nil
	})

	var p *Promise
	p = initial.Then(func(any) (any, error) {
		return p, nil
	}, nil)

	p.Then(func(v any) (any, error) {
		t.Errorf("Promise should be rejected due to cycle detection, but was fulfilled with %v", v)
		return nil, nil
	}, func(val error) (any, error) {
		if val.Error() != "TypeError: Chaining cycle detected for promise" {
			t.Errorf("Expected cycle detection error message, got %s", val.Error())
		}
		return nil, nil
	})
}

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

	time.Sleep(time.Second)
}

// 测试执行器中多次调用resolve或reject
func TestExecutorMultipleCalls(t *testing.T) {
	t.Parallel()
	slowProm := el.NewPromise(func(resolve, reject func(v any)) error {
		el.SetTimeout(func() {
			resolve("slow")
		}, 200)
		return nil
	})
	fastProm := el.NewPromise(func(resolve, reject func(v any)) error {
		el.SetTimeout(func() {
			resolve("fast")
		}, 100)
		return nil
	})

	p := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve(slowProm)
		reject(fastProm)
		return nil
	})

	p.Then(func(v any) (any, error) {
		if v != "slow" {
			t.Errorf("Expected value 'slow', got %v", v)
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise should not be rejected, got reason: %v", v)
		return nil, nil
	})

	time.Sleep(time.Second)
}

// 测试nil执行器
func TestExecutorNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for nil executor")
		}
	}()

	el.NewPromise(nil)
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

	time.Sleep(time.Second)
}
