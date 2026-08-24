package promise_test

import (
	"errors"
	"testing"
	"time"
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

	time.Sleep(2 * time.Second)
}
