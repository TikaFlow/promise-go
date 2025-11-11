package promise_test

import (
	"errors"
	"github.com/TikaFlow/promise-go"
	ip "github.com/TikaFlow/promise-go/ipromise"
	"testing"
	"time"
)

// 测试Promise的基本创建和解决
func TestPromiseBasicResolve(t *testing.T) {
	p := promise.New(func(resolve, reject func(any)) error {
		resolve("success")
		return nil
	})

	// 使用Then方法来验证Promise的状态和值，模仿JavaScript Promise的使用方式
	p.Then(func(v any) (any, error) {
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise should not be rejected, got reason: %v", v)
		return nil, nil
	})
}

// 测试Promise的基本创建和拒绝
func TestPromiseBasicReject(t *testing.T) {
	p := promise.New(func(resolve, reject func(any)) error {
		reject("failure")
		return nil
	})

	// 使用Then方法验证Promise被拒绝的状态和值
	p.Then(func(v any) (any, error) {
		t.Errorf("Promise should not be fulfilled, got value: %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "failure" {
			t.Errorf("Expected rejection reason 'failure', got %v", v)
		}
		return nil, nil
	})
}

// 测试Promise执行器错误处理
func TestPromiseExecutorError(t *testing.T) {
	errorMsg := "executor error"

	p := promise.New(func(resolve, reject func(any)) error {
		return errors.New(errorMsg)
	})

	// 使用Then方法验证执行器错误被正确捕获并拒绝Promise
	p.Then(func(v any) (any, error) {
		t.Errorf("Promise should not be fulfilled, got value: %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if val, ok := v.(error); ok {
			if val.Error() != errorMsg {
				t.Errorf("Expected error '%s', got '%s'", errorMsg, val.Error())
			}
		} else {
			t.Errorf("Expected error type, got %T", v)
		}
		return nil, nil
	})
}

// 测试nil执行器
func TestPromiseNilExecutor(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for nil executor")
		}
	}()

	promise.New(nil)
	// 注意：由于执行器为nil会立即panic，所以不需要RunLoop和等待
}

// 测试执行器中多次调用resolve或reject
func TestPromiseExecutorMultipleCalls(t *testing.T) {
	slowProm := promise.New(func(resolve, reject func(any)) error {
		go func() {
			time.Sleep(200 * time.Millisecond)
			resolve("slow")
		}()
		return nil
	})
	fastProm := promise.New(func(resolve, reject func(any)) error {
		go func() {
			time.Sleep(100 * time.Millisecond)
			resolve("fast")
		}()
		return nil
	})

	p := promise.New(func(resolve, reject func(any)) error {
		resolve(slowProm)
		reject(fastProm)
		return nil
	})

	p.Then(func(v any) (any, error) {
		if v != "slow" {
			t.Errorf("Expected value 'slow', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise should not be rejected, got reason: %v", v)
		return nil, nil
	})
}

// 测试执行器在resolve或reject已调用后报错
func TestPromiseExecutorErrorAfterResolved(t *testing.T) {
	delayResolve := promise.New(func(resolve, reject func(any)) error {
		go func() {
			time.Sleep(100 * time.Millisecond)
			resolve("success")
		}()
		return nil
	})

	p := promise.New(func(resolve, reject func(any)) error {
		resolve(delayResolve)
		return errors.New("executor error after resolve")
	})

	p.Then(func(v any) (any, error) {
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise should not be rejected, got reason: %v", v)
		return nil, nil
	})
}

// 测试Then方法的基本功能 - 成功回调
func TestPromiseThenFulfilled(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		resolve(1)
		return nil
	})

	p1.Then(func(v any) (any, error) {
		return v.(int) + 1, nil
	}, nil).Then(func(v any) (any, error) {
		if v != 2 {
			t.Errorf("Expected value 2 after then, got %v", v)
		}
		return nil, nil
	}, nil)
}

// 测试Then方法的基本功能 - 拒绝回调
func TestPromiseThenRejected(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		reject("error")
		return nil
	})

	p1.Then(nil, func(v any) (any, error) {
		return "handled: " + v.(string), nil
	}).Then(func(v any) (any, error) {
		if v != "handled: error" {
			t.Errorf("Expected value 'handled: error', got %v", v)
		}
		return nil, nil
	}, nil)
}

// 测试Then方法的穿透 - 成功状态
func TestPromiseThenPassThroughFulfilled(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		resolve("value")
		return nil
	})

	p1.Then(nil, nil).Then(func(v any) (any, error) {
		if v != "value" {
			t.Errorf("Expected value 'value' after pass through, got %v", v)
		}
		return nil, nil
	}, nil)
}

// 测试Then方法的穿透 - 拒绝状态
func TestPromiseThenPassThroughRejected(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		reject("reason")
		return nil
	})

	p1.Then(nil, nil).Then(nil, func(v any) (any, error) {
		if v != "reason" {
			t.Errorf("Expected rejection reason 'reason' after pass through, got %v", v)
		}
		return nil, nil
	})
}

// 测试Then方法回调函数抛出错误
func TestPromiseThenCallbackError(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		resolve("value")
		return nil
	})

	p1.Then(func(v any) (any, error) {
		return "callback error", errors.New("error")
	}, nil).Then(nil, func(v any) (any, error) {
		if v != "callback error" {
			t.Errorf("Expected error value 'callback error', got %v", v)
		}
		return nil, nil
	})

	p2 := promise.Reject("error")
	p2.Then(nil, func(v any) (any, error) {
		return "callback error", errors.New("error")
	}).Then(nil, func(v any) (any, error) {
		if v != "callback error" {
			t.Errorf("Expected error value 'callback error', got %v", v)
		}
		return nil, nil
	})
}

// 测试Then方法链式调用
func TestPromiseThenChaining(t *testing.T) {
	promise.New(func(resolve, reject func(any)) error {
		resolve(1)
		return nil
	}).Then(func(v any) (any, error) {
		return v.(int) + 1, nil
	}, nil).Then(func(v any) (any, error) {
		return v.(int) * 2, nil
	}, nil).Then(func(v any) (any, error) {
		if v != 4 {
			t.Errorf("Expected final value 4 after chaining, got %v", v)
		}
		return nil, nil
	}, nil)
}

// 测试Catch方法
func TestPromiseCatch(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		reject("error")
		return nil
	})

	p1.Catch(func(v any) (any, error) {
		return "caught: " + v.(string), nil
	}).Then(func(v any) (any, error) {
		if v != "caught: error" {
			t.Errorf("Expected value 'caught: error', got %v", v)
		}
		return nil, nil
	}, nil)
}

// 测试Finally方法 - 成功状态
func TestPromiseFinallyFulfilled(t *testing.T) {
	finallyCalled := false

	p1 := promise.New(func(resolve, reject func(any)) error {
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
func TestPromiseFinallyRejected(t *testing.T) {
	finallyCalled := false

	p1 := promise.New(func(resolve, reject func(any)) error {
		reject("error")
		return nil
	})

	p1.Finally(func() (any, error) {
		finallyCalled = true
		return nil, nil
	}).Then(nil, func(v any) (any, error) {
		if !finallyCalled {
			t.Errorf("Finally callback was not called")
		}
		if v != "error" {
			t.Errorf("Expected value 'error', got %v", v)
		}
		return nil, nil
	})
}

// 测试Finally方法抛出错误
func TestPromiseFinallyError(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		resolve("success")
		return nil
	})

	p1.Finally(func() (any, error) {
		return "finally error", errors.New("error")
	}).Then(func(v any) (any, error) {
		t.Errorf("Expected promise to be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "finally error" {
			t.Errorf("Expected error value 'finally error', got %v", v)
		}
		return nil, nil
	})
}

// 测试Finally方法返回被拒绝的Promise
func TestPromiseFinallyReturnsRejectedPromise(t *testing.T) {
	rejectedPromise := promise.New(func(resolve, reject func(any)) error {
		reject("rejected from finally")
		return nil
	})

	p1 := promise.New(func(resolve, reject func(any)) error {
		resolve("success")
		return nil
	})

	p1.Finally(func() (any, error) {
		return rejectedPromise, nil
	}).Then(func(v any) (any, error) {
		t.Errorf("Expected promise to be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "rejected from finally" {
			t.Errorf("Expected rejection reason 'rejected from finally', got %v", v)
		}
		return nil, nil
	})
}

// 测试All方法 - 所有Promise都成功
func TestPromiseAllFulfilled(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		resolve(1)
		return nil
	})
	p2 := promise.New(func(resolve, reject func(any)) error {
		resolve(2)
		return nil
	})
	p3 := promise.New(func(resolve, reject func(any)) error {
		resolve(3)
		return nil
	})

	promise.All([]ip.Promise{p1, p2, p3}).Then(func(v any) (any, error) {
		results, ok := v.([]any)
		if !ok {
			t.Errorf("Expected []any type, got %T", v)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
		if results[0] != 1 || results[1] != 2 || results[2] != 3 {
			t.Errorf("Expected [1, 2, 3], got %v", results)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.All should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})
}

// 测试All方法 - 有一个Promise被拒绝
func TestPromiseAllRejected(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		resolve(1)
		return nil
	})
	p2 := promise.New(func(resolve, reject func(any)) error {
		reject("error")
		return nil
	})
	p3 := promise.New(func(resolve, reject func(any)) error {
		resolve(3)
		return nil
	})

	promise.All([]ip.Promise{p1, p2, p3}).Then(func(v any) (any, error) {
		t.Errorf("Promise.All should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "error" {
			t.Errorf("Expected rejection value 'error', got %v", v)
		}
		return nil, nil
	})
}

// 测试All方法 - 空数组
func TestPromiseAllEmptyArray(t *testing.T) {
	allPromise := promise.All([]ip.Promise{})

	allPromise.Then(func(v any) (any, error) {
		results, ok := v.([]any)
		if !ok {
			t.Errorf("Expected []any type, got %T", v)
		}
		if len(results) != 0 {
			t.Errorf("Expected empty array, got %d elements", len(results))
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.All should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})
}

// 测试All方法 - nil数组
func TestPromiseAllNilArray(t *testing.T) {
	promise.All(nil).Then(func(v any) (any, error) {
		t.Errorf("Promise.All with nil array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "TypeError: nil is not iterable" {
			t.Errorf("Expected error value 'TypeError: nil is not iterable', got %v", v)
		}
		return nil, nil
	})
}

// 测试AllSettled方法
func TestPromiseAllSettled(t *testing.T) {

	p1 := promise.New(func(resolve, reject func(any)) error {
		resolve(1)
		return nil
	})
	p2 := promise.New(func(resolve, reject func(any)) error {
		reject("error")
		return nil
	})
	p3 := promise.New(func(resolve, reject func(any)) error {
		resolve(3)
		return nil
	})

	promise.AllSettled([]ip.Promise{p1, p2, p3}).Then(func(v any) (any, error) {
		results, ok := v.([]map[string]any)
		if !ok {
			t.Errorf("Expected []map[string]any type, got %T", v)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		if results[0]["status"] != ip.Fulfilled || results[0]["value"] != 1 {
			t.Errorf("Expected first result to be fulfilled with value 1, got %v", results[0])
		}
		if results[1]["status"] != ip.Rejected || results[1]["reason"] != "error" {
			t.Errorf("Expected second result to be rejected with reason 'error', got %v", results[1])
		}
		if results[2]["status"] != ip.Fulfilled || results[2]["value"] != 3 {
			t.Errorf("Expected third result to be fulfilled with value 3, got %v", results[2])
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.AllSettled should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})
}

// 测试AllSettled方法 - 空数组
func TestPromiseAllSettledEmptyArray(t *testing.T) {
	promise.AllSettled([]ip.Promise{}).Then(func(v any) (any, error) {
		results, ok := v.([]map[string]any)
		if !ok {
			t.Errorf("Expected []map[string]any type, got %T", v)
		}
		if len(results) != 0 {
			t.Errorf("Expected empty array, got %d elements", len(results))
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.AllSettled should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})
}

// 测试AllSettled方法 - nil数组
func TestPromiseAllSettledNilArray(t *testing.T) {
	promise.AllSettled(nil).Then(func(v any) (any, error) {
		t.Errorf("Promise.AllSettled with nil array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "TypeError: nil is not iterable" {
			t.Errorf("Expected error value 'TypeError: nil is not iterable', got %v", v)
		}
		return nil, nil
	})
}

// 测试Any方法 - 有一个Promise成功
func TestPromiseAnyFulfilled(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		reject("error1")
		return nil
	})
	p2 := promise.New(func(resolve, reject func(any)) error {
		resolve("success")
		return nil
	})
	p3 := promise.New(func(resolve, reject func(any)) error {
		reject("error2")
		return nil
	})

	promise.Any([]ip.Promise{p1, p2, p3}).Then(func(v any) (any, error) {
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.Any should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})
}

// 测试Any方法 - 所有Promise都失败
func TestPromiseAnyAllRejected(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		reject("error1")
		return nil
	})
	p2 := promise.New(func(resolve, reject func(any)) error {
		reject("error2")
		return nil
	})

	promise.Any([]ip.Promise{p1, p2}).Then(func(v any) (any, error) {
		t.Errorf("Promise.Any should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		agg, ok := v.(map[string]any)
		if !ok {
			t.Errorf("Expected map[string]any, got %T", v)
			return nil, nil
		}
		if len(agg["errors"].([]any)) != 2 {
			t.Errorf("Expected 2 errors, got %d", len(agg["errors"].([]any)))
			return nil, nil
		}
		if agg["errors"].([]any)[0] != "error1" || agg["errors"].([]any)[1] != "error2" {
			t.Errorf("Expected errors ['error1','error2'], got %v", agg["errors"].([]any))
		}
		return nil, nil
	})
}

// 测试Any方法 - 空数组
func TestPromiseAnyEmptyArray(t *testing.T) {
	promise.Any([]ip.Promise{}).Then(func(v any) (any, error) {
		t.Errorf("Promise.Any with empty array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		agg, ok := v.(map[string]any)
		if !ok {
			t.Errorf("Expected map[string]any, got %T", v)
			return nil, nil
		}
		if len(agg["errors"].([]any)) != 0 {
			t.Errorf("Expected 0 errors, got %d", len(agg["errors"].([]any)))
			return nil, nil
		}
		return nil, nil
	})
}

// 测试Any方法 - nil数组
func TestPromiseAnyNilArray(t *testing.T) {
	promise.Any(nil).Then(func(v any) (any, error) {
		t.Errorf("Promise.Any with nil array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "TypeError: nil is not iterable" {
			t.Errorf("Expected error value 'TypeError: nil is not iterable', got %v", v)
		}
		return nil, nil
	})
}

// 测试Race方法 - 第一个完成的是成功的Promise
func TestPromiseRaceFulfilled(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		// 使用goroutine模拟异步延迟
		go func() {
			// 等待时间较长
			time.Sleep(100 * time.Millisecond)
			reject("error")
		}()
		return nil
	})
	p2 := promise.New(func(resolve, reject func(any)) error {
		// 使用goroutine模拟异步延迟
		go func() {
			// 等待时间较短
			time.Sleep(50 * time.Millisecond)
			resolve("success")
		}()
		return nil
	})

	promise.Race([]ip.Promise{p1, p2}).Then(func(v any) (any, error) {
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.Race should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})
}

// 测试Race方法 - 第一个完成的是失败的Promise
func TestPromiseRaceRejected(t *testing.T) {
	p1 := promise.New(func(resolve, reject func(any)) error {
		// 使用goroutine模拟异步延迟
		go func() {
			// 等待时间较短
			time.Sleep(50 * time.Millisecond)
			reject("error")
		}()
		return nil
	})
	p2 := promise.New(func(resolve, reject func(any)) error {
		// 使用goroutine模拟异步延迟
		go func() {
			// 等待时间较长
			time.Sleep(100 * time.Millisecond)
			resolve("success")
		}()
		return nil
	})

	promise.Race([]ip.Promise{p1, p2}).Then(func(v any) (any, error) {
		t.Errorf("Promise.Race should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "error" {
			t.Errorf("Expected value 'error', got %v", v)
		}
		return nil, nil
	})
}

// 测试Race方法 - 空数组
func TestPromiseRaceEmptyArray(t *testing.T) {
	racePromise := promise.Race([]ip.Promise{})

	go func() {
		// 空数组的Race应该返回一个pending状态的Promise
		// 这里我们验证它仍然是pending状态
		var state string
		// 设置超时
		timer := time.NewTimer(100 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-racePromise.Done():
			state = racePromise.State()
		case <-timer.C:
			// 超时，说明Promise仍然是pending状态
			state = ip.Pending
		}

		if state != ip.Pending {
			t.Errorf("Expected state Pending for empty array Race, got %s", state)
		}
	}()
}

// 测试Race方法 - nil数组
func TestPromiseRaceNilArray(t *testing.T) {
	promise.Race(nil).Then(func(v any) (any, error) {
		t.Errorf("Promise.Race with nil array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "TypeError: nil is not iterable" {
			t.Errorf("Expected error value 'TypeError: nil is not iterable', got %v", v)
		}
		return nil, nil
	})
}

// 测试Resolve方法 - 普通值
func TestPromiseResolve(t *testing.T) {
	promise.Resolve("value").Then(func(v any) (any, error) {
		if v != "value" {
			t.Errorf("Expected value 'value', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.Resolve should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})
}

// 测试Resolve方法 - Promise对象
func TestPromiseResolvePromise(t *testing.T) {
	original := promise.New(func(resolve, reject func(any)) error {
		resolve("original")
		return nil
	})

	p := promise.Resolve(original)

	if p != original {
		t.Errorf("Expected Resolve to return the same Promise instance")
	}

	p.Then(func(v any) (any, error) {
		if v != "original" {
			t.Errorf("Expected value 'original', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Expected state Fulfilled, got Rejected with %v", v)
		return nil, nil
	})
}

// 测试Reject方法
func TestPromiseReject(t *testing.T) {
	promise.Reject("reason").Then(func(v any) (any, error) {
		t.Errorf("Promise.Reject should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "reason" {
			t.Errorf("Expected value 'reason', got %v", v)
		}
		return nil, nil
	})
}

// 测试Try方法 - 成功
func TestPromiseTrySuccess(t *testing.T) {
	promise.Try(func(args ...any) (any, error) {
		return "success", nil
	}).Then(func(v any) (any, error) {
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.Try should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})
}

// 测试Try方法 - 失败
func TestPromiseTryError(t *testing.T) {
	promise.Try(func(args ...any) (any, error) {
		return "error value", errors.New("error")
	}).Then(func(v any) (any, error) {
		t.Errorf("Promise.Try should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "error value" {
			t.Errorf("Expected error value 'error value', got %v", v)
		}
		return nil, nil
	})
}

// 测试Try方法 - nil函数
func TestPromiseTryNilFunc(t *testing.T) {
	promise.Try(nil).Then(func(v any) (any, error) {
		t.Errorf("Promise.Try with nil func should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if val, ok := v.(string); ok {
			if val != "Promise executor must be a function" {
				t.Errorf("Expected error message, got %s", val)
			}
		} else {
			t.Errorf("Expected string error message, got %T", v)
		}
		return nil, nil
	})
}

// 测试PromiseWithResolvers方法
func TestPromiseWithResolvers(t *testing.T) {
	// 测试成功情况
	p, resolve, _ := promise.PromiseWithResolvers()
	resolve("resolved value")

	p.Then(func(v any) (any, error) {
		if v != "resolved value" {
			t.Errorf("Expected value 'resolved value', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})

	// 测试拒绝情况
	p2, _, reject2 := promise.PromiseWithResolvers()
	reject2("rejected reason")

	p2.Then(func(v any) (any, error) {
		t.Errorf("Promise should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "rejected reason" {
			t.Errorf("Expected value 'rejected reason', got %v", v)
		}
		return nil, nil
	})
}

// 测试循环引用检测
func TestPromiseCycleDetection(t *testing.T) {
	initial := promise.New(func(resolve, reject func(any)) error {
		resolve("initial")
		return nil
	})

	var p ip.Promise
	p = initial.Then(func(_ any) (any, error) {
		return p, nil
	}, nil)

	p.Then(func(v any) (any, error) {
		t.Errorf("Promise should be rejected due to cycle detection, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if val, ok := v.(string); ok {
			if val != "TypeError: Chaining cycle detected for promise" {
				t.Errorf("Expected cycle detection error message, got %s", val)
			}
		} else {
			t.Errorf("Expected string error message, got %T", v)
		}
		return nil, nil
	})
}

// 测试Thenable对象处理
func TestPromiseThenable(t *testing.T) {
	thenable := promise.New(func(resolve, reject func(any)) error {
		resolve("thenable value")
		return nil
	})

	p := promise.New(func(resolve, reject func(any)) error {
		resolve(thenable)
		return nil
	})

	p.Then(func(v any) (any, error) {
		if v != "thenable value" {
			t.Errorf("Expected value 'thenable value', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise with Thenable should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})
}

// 测试多个Then调用的顺序
func TestPromiseMultipleThenOrder(t *testing.T) {
	p := promise.New(func(resolve, reject func(any)) error {
		resolve("value")
		return nil
	})

	var results []string

	p.Then(func(v any) (any, error) {
		results = append(results, "then1")
		return v, nil
	}, nil)

	p.Then(func(v any) (any, error) {
		results = append(results, "then2")
		return v, nil
	}, nil)

	p.Then(func(v any) (any, error) {
		results = append(results, "then3")
		return v, nil
	}, nil)

	expected := []string{"then1", "then2", "then3"}
	p.Finally(func() (any, error) {
		if len(results) != len(expected) {
			t.Errorf("Expected %d results, got %d", len(expected), len(results))
		} else {
			for i := range results {
				if results[i] != expected[i] {
					t.Errorf("Expected result %d to be '%s', got '%s'", i, expected[i], results[i])
				}
			}
		}
		return nil, nil
	})
}

// 测试 SetTimeout 函数
func TestSetTimeout(t *testing.T) {
	var str string
	promise.SetTimeout(func() {
		str = "timeout value"
	}, 100)

	time.Sleep(80 * time.Millisecond)
	if str != "" {
		t.Errorf("Expected str '', got %s", str)
	}

	time.Sleep(40 * time.Millisecond)
	if str != "timeout value" {
		t.Errorf("Expected str 'timeout value', got %s", str)
	}
}

// 测试 SetTimeout 函数 - 取消
func TestSetTimeoutCancel(t *testing.T) {
	p := promise.New(func(resolve, reject func(any)) error {
		id := promise.SetTimeout(func() {
			resolve("timeout value")
		}, 100)
		promise.ClearTimeout(id)
		return nil
	})

	time.Sleep(120 * time.Millisecond)
	if p.State() != ip.Pending {
		t.Errorf("Expected state Pending, got %v", p.State())
	}
}

// 测试 SetTimeout 函数 - 毫秒数为负数
func TestSetTimeoutNegativeMillis(t *testing.T) {
	var str string
	promise.SetTimeout(func() {
		str = "timeout value"
	}, -100)

	time.Sleep(80 * time.Millisecond)
	if str != "" {
		t.Errorf("Expected str '', got %s", str)
	}

	time.Sleep(40 * time.Millisecond)
	if str != "timeout value" {
		t.Errorf("Expected str 'timeout value', got %s", str)
	}
}

// 测试 SetTimeout 函数 - 毫秒数为 0
func TestSetTimeoutZeroMillis(t *testing.T) {
	var str string
	promise.SetTimeout(func() {
		str = "timeout value"
	}, 0)

	time.Sleep(20 * time.Millisecond)
	if str != "timeout value" {
		t.Errorf("Expected str 'timeout value', got %s", str)
	}
}

// 测试 SetTimeout 函数 - 长延迟
func TestSetTimeoutLongDelay(t *testing.T) {
	var str string
	promise.SetTimeout(func() {
		str = "timeout value"
	}, 1000)

	time.Sleep(980 * time.Millisecond)
	if str != "" {
		t.Errorf("Expected str '', got %s", str)
	}

	time.Sleep(40 * time.Millisecond)
	if str != "timeout value" {
		t.Errorf("Expected str 'timeout value', got %s", str)
	}
}

// 测试 SetInterval 函数
func TestSetInterval(t *testing.T) {
	var str string
	var count int
	var ch chan struct{}

	ch = promise.SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			promise.ClearInterval(ch)
		}
	}, 200)

	time.Sleep(20 * time.Millisecond)
	time.Sleep(160 * time.Millisecond)
	if str != "" {
		t.Errorf("Expected str '', got %s", str)
	}
	time.Sleep(40 * time.Millisecond)
	if str != "interval " {
		t.Errorf("Expected str 'interval ', got %s", str)
	}

	time.Sleep(160 * time.Millisecond)
	if str != "interval " {
		t.Errorf("Expected str 'interval ', got %s", str)
	}
	time.Sleep(40 * time.Millisecond)
	if str != "interval interval " {
		t.Errorf("Expected str 'interval interval ', got %s", str)
	}

	time.Sleep(160 * time.Millisecond)
	if str != "interval interval " {
		t.Errorf("Expected str 'interval interval ', got %s", str)
	}
	time.Sleep(40 * time.Millisecond)
	if str != "interval interval interval " {
		t.Errorf("Expected str 'interval interval interval ', got %s", str)
	}
}

// 测试 SetInterval 函数 - 取消 - 第1次执行
func TestSetIntervalCancelFirst(t *testing.T) {
	var str string
	var count int
	var ch chan struct{}

	ch = promise.SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			promise.ClearInterval(ch)
		}
	}, 200)

	time.Sleep(20 * time.Millisecond)
	promise.ClearInterval(ch)
	time.Sleep(200 * time.Millisecond)
	if str != "" {
		t.Errorf("Expected str '', got %s", str)
	}
}

// 测试 SetInterval 函数 - 取消 - 非首次执行
func TestSetIntervalCancelNonFirst(t *testing.T) {
	var str string
	var count int
	var ch chan struct{}

	ch = promise.SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			promise.ClearInterval(ch)
		}
	}, 200)

	time.Sleep(20 * time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	if str != "interval " {
		t.Errorf("Expected str 'interval ', got %s", str)
	}
	promise.ClearInterval(ch)
	time.Sleep(400 * time.Millisecond)
	if str != "interval " {
		t.Errorf("Expected str 'interval ', got %s", str)
	}
}

// 测试 SetInterval 函数 - 长延迟
func TestSetIntervalLongDelay(t *testing.T) {
	var str string
	var count int
	var ch chan struct{}

	ch = promise.SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			promise.ClearInterval(ch)
		}
	}, 1000)

	time.Sleep(20 * time.Millisecond)
	time.Sleep(960 * time.Millisecond)
	if str != "" {
		t.Errorf("Expected str '', got %s", str)
	}
	time.Sleep(40 * time.Millisecond)
	if str != "interval " {
		t.Errorf("Expected str 'interval ', got %s", str)
	}

	time.Sleep(960 * time.Millisecond)
	if str != "interval " {
		t.Errorf("Expected str 'interval ', got %s", str)
	}
	time.Sleep(40 * time.Millisecond)
	if str != "interval interval " {
		t.Errorf("Expected str 'interval interval ', got %s", str)
	}

	time.Sleep(960 * time.Millisecond)
	if str != "interval interval " {
		t.Errorf("Expected str 'interval interval ', got %s", str)
	}
	time.Sleep(40 * time.Millisecond)
	if str != "interval interval interval " {
		t.Errorf("Expected str 'interval interval interval ', got %s", str)
	}
}

// 测试异步调用顺序
// FIXME: 测试结果与预期不符
func TestAsyncCallOrder(t *testing.T) {
	// 创建结果字符串，用于记录执行顺序
	var result string
	// 用于追加结果的辅助函数
	appendResult := func(index string) {
		if result == "" {
			result = "[" + index + "]"
		} else {
			result += "-[" + index + "]"
		}
	}

	// 同步代码
	appendResult("01")

	// setTimeout - 宏任务 1
	promise.SetTimeout(func() {
		appendResult("28")

		// Promise.reject().catch - 微任务 1-1
		promise.Reject("error").Catch(func(v any) (any, error) {
			appendResult("29")
			return nil, nil
		})

		// queueMicrotask - 微任务 1-2
		promise.QueueMicrotask(func() {
			appendResult("30")
			// setTimeout - 宏任务 1-2-1
			promise.SetTimeout(func() {
				appendResult("37")
			}, 0)
		})

		// setTimeout - 宏任务 1-3
		promise.SetTimeout(func() {
			appendResult("41")
		}, 30)
	}, 50)

	// Promise.resolve() - p1
	p1 := promise.Resolve(nil)
	p1.Then(func(v any) (any, error) {
		appendResult("04")

		// Promise.resolve().then - 微任务 2-1
		promise.Resolve(nil).Then(func(v any) (any, error) {
			appendResult("10")
			return nil, nil
		}, nil)

		// queueMicrotask - 微任务 2-2
		promise.QueueMicrotask(func() {
			appendResult("11")
			// setTimeout - 宏任务 2-2-1
			promise.SetTimeout(func() {
				appendResult("38")
			}, 50)
		})

		// setTimeout - 宏任务 2-3
		promise.SetTimeout(func() {
			appendResult("26")
		}, 0)

		return nil, nil
	}, nil)

	// Promise.resolve() - p2
	p2 := promise.Resolve(nil)
	p2.Then(func(v any) (any, error) {
		appendResult("05")
		return nil, nil
	}, nil).Then(func(v any) (any, error) {
		appendResult("12")

		// Promise.resolve().then().then() - 微任务 3-2-1 和 3-2-2
		promise.Resolve(nil).Then(func(v any) (any, error) {
			appendResult("19")
			return nil, nil
		}, nil).Then(func(v any) (any, error) {
			appendResult("24")
			return nil, nil
		}, nil)

		// queueMicrotask - 微任务 3-2-3
		promise.QueueMicrotask(func() {
			appendResult("20")
		})

		// setTimeout - 宏任务 3-2-4
		promise.SetTimeout(func() {
			appendResult("39")

			// queueMicrotask - 微任务 3-2-4-1
			promise.QueueMicrotask(func() {
				appendResult("40")
			})
		}, 50)

		return nil, nil
	}, nil).Then(func(v any) (any, error) {
		appendResult("21")
		return nil, nil
	}, nil)

	// new Promise - p3
	p3 := promise.New(func(resolve, reject func(any)) error {
		appendResult("02")
		resolve(4)
		return nil
	})

	// p3.then - 微任务 4-1
	p3.Then(func(v any) (any, error) {
		appendResult("06")

		// Promise.reject().catch - 微任务 4-1-1
		promise.Reject("error").Catch(func(v any) (any, error) {
			appendResult("13")

			// Promise.resolve().then - 微任务 4-1-1-1
			promise.Resolve(nil).Then(func(v any) (any, error) {
				appendResult("22")
				return nil, nil
			}, nil)

			return nil, nil
		})

		// queueMicrotask - 微任务 4-1-2
		promise.QueueMicrotask(func() {
			appendResult("14")
		})

		// setTimeout - 宏任务 4-1-3
		promise.SetTimeout(func() {
			appendResult("35")
		}, 50)

		return nil, nil
	}, nil)

	// p3.then - 微任务 4-2
	p3.Then(func(v any) (any, error) {
		appendResult("07")

		// Promise.resolve().finally - 微任务 4-2-1
		promise.Resolve(nil).Finally(func() (any, error) {
			appendResult("15")
			return nil, nil
		})

		// queueMicrotask - 微任务 4-2-2
		promise.QueueMicrotask(func() {
			appendResult("16")
		})

		// setTimeout - 宏任务 4-2-3
		promise.SetTimeout(func() {
			appendResult("36")
		}, 50)

		return nil, nil
	}, nil)

	// p3.then - 微任务 4-3
	p3.Then(func(v any) (any, error) {
		appendResult("08")
		return nil, nil
	}, nil)

	// setInterval - 宏任务 5
	count := 1
	var intervalCh chan struct{}
	intervalCh = promise.SetInterval(func() {
		appendResult("31")

		if count >= 2 {
			promise.ClearInterval(intervalCh)
		}
		count++

		// Promise.resolve().finally - 微任务 5-1
		promise.Resolve(nil).Finally(func() (any, error) {
			appendResult("32")

			// Promise.resolve().finally - 微任务 5-1-1
			promise.Resolve(nil).Finally(func() (any, error) {
				appendResult("34")
				return nil, nil
			})

			return nil, nil
		})

		// queueMicrotask - 微任务 5-2
		promise.QueueMicrotask(func() {
			appendResult("33")
		})

		// setTimeout - 宏任务 5-3
		promise.SetTimeout(func() {
			appendResult("42")
		}, 30)
	}, 50)

	// queueMicrotask - 微任务 6
	promise.QueueMicrotask(func() {
		appendResult("09")

		// Promise.resolve().then().then().then() - 微任务 6-1, 6-2, 6-3
		p := promise.Resolve(nil)
		p.Then(func(v any) (any, error) {
			appendResult("17")
			return nil, nil
		}, nil).Then(func(v any) (any, error) {
			appendResult("23")
			return nil, nil
		}, nil).Then(func(v any) (any, error) {
			appendResult("25")
			return nil, nil
		}, nil)

		// queueMicrotask - 微任务 6-4
		promise.QueueMicrotask(func() {
			appendResult("18")
		})

		// setTimeout - 宏任务 6-5
		promise.SetTimeout(func() {
			appendResult("27")
		}, 0)
	})

	// 同步代码结束
	appendResult("03")

	// 等待所有异步任务完成
	time.Sleep(360 * time.Millisecond)

	// 验证结果是否符合预期
	expected := "[01]-[02]-[03]-[04]-[05]-[06]-[07]-[08]-[09]-[10]-[11]-[12]-[13]-[14]-[15]-[16]-[17]-[18]-[19]-[20]-[21]-[22]-[23]-[24]-[25]-[26]-[27]-[28]-[29]-[30]-[31]-[32]-[33]-[34]-[35]-[36]-[38]-[39]-[40]-[37]-[41]-[42]-[31]-[32]-[33]-[34]-[42]"
	if result != expected {
		t.Errorf("Expected result:\n%s\n\nGot:\n%s", expected, result)
	}
}

func TestMain(m *testing.M) {
	m.Run()
}
