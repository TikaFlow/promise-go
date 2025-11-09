package promise_test

import (
	"errors"
	"github.com/TikaFlow/promise-go"
	ip "github.com/TikaFlow/promise-go/ipromise"
	"testing"
	"time"
)

var (
	pm *promise.PromiseManager
)

// 测试Promise的基本创建和解决
func TestPromiseBasicResolve(t *testing.T) {
	p := pm.New(func(resolve, reject func(any)) error {
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
	p := pm.New(func(resolve, reject func(any)) error {
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

	p := pm.New(func(resolve, reject func(any)) error {
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

	pm.New(nil)
	// 注意：由于执行器为nil会立即panic，所以不需要RunLoop和等待
}

// 测试Then方法的基本功能 - 成功回调
func TestPromiseThenFulfilled(t *testing.T) {
	p1 := pm.New(func(resolve, reject func(any)) error {
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
	p1 := pm.New(func(resolve, reject func(any)) error {
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
	p1 := pm.New(func(resolve, reject func(any)) error {
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
	p1 := pm.New(func(resolve, reject func(any)) error {
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
	p1 := pm.New(func(resolve, reject func(any)) error {
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
}

// 测试Then方法链式调用
func TestPromiseThenChaining(t *testing.T) {
	pm.New(func(resolve, reject func(any)) error {
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
	p1 := pm.New(func(resolve, reject func(any)) error {
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

	p1 := pm.New(func(resolve, reject func(any)) error {
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

	p1 := pm.New(func(resolve, reject func(any)) error {
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
	p1 := pm.New(func(resolve, reject func(any)) error {
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
	rejectedPromise := pm.New(func(resolve, reject func(any)) error {
		reject("rejected from finally")
		return nil
	})

	p1 := pm.New(func(resolve, reject func(any)) error {
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
	p1 := pm.New(func(resolve, reject func(any)) error {
		resolve(1)
		return nil
	})
	p2 := pm.New(func(resolve, reject func(any)) error {
		resolve(2)
		return nil
	})
	p3 := pm.New(func(resolve, reject func(any)) error {
		resolve(3)
		return nil
	})

	pm.All([]ip.Promise{p1, p2, p3}).Then(func(v any) (any, error) {
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
	p1 := pm.New(func(resolve, reject func(any)) error {
		resolve(1)
		return nil
	})
	p2 := pm.New(func(resolve, reject func(any)) error {
		reject("error")
		return nil
	})
	p3 := pm.New(func(resolve, reject func(any)) error {
		resolve(3)
		return nil
	})

	pm.All([]ip.Promise{p1, p2, p3}).Then(func(v any) (any, error) {
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
	allPromise := pm.All([]ip.Promise{})

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

// 测试AllSettled方法
func TestPromiseAllSettled(t *testing.T) {

	p1 := pm.New(func(resolve, reject func(any)) error {
		resolve(1)
		return nil
	})
	p2 := pm.New(func(resolve, reject func(any)) error {
		reject("error")
		return nil
	})
	p3 := pm.New(func(resolve, reject func(any)) error {
		resolve(3)
		return nil
	})

	pm.AllSettled([]ip.Promise{p1, p2, p3}).Then(func(v any) (any, error) {
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
	pm.AllSettled([]ip.Promise{}).Then(func(v any) (any, error) {
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

// 测试Any方法 - 有一个Promise成功
func TestPromiseAnyFulfilled(t *testing.T) {
	p1 := pm.New(func(resolve, reject func(any)) error {
		reject("error1")
		return nil
	})
	p2 := pm.New(func(resolve, reject func(any)) error {
		resolve("success")
		return nil
	})
	p3 := pm.New(func(resolve, reject func(any)) error {
		reject("error2")
		return nil
	})

	pm.Any([]ip.Promise{p1, p2, p3}).Then(func(v any) (any, error) {
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
	p1 := pm.New(func(resolve, reject func(any)) error {
		reject("error1")
		return nil
	})
	p2 := pm.New(func(resolve, reject func(any)) error {
		reject("error2")
		return nil
	})

	pm.Any([]ip.Promise{p1, p2}).Then(func(v any) (any, error) {
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
	pm.Any([]ip.Promise{}).Then(func(v any) (any, error) {
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

// 测试Race方法 - 第一个完成的是成功的Promise
func TestPromiseRaceFulfilled(t *testing.T) {
	p1 := pm.New(func(resolve, reject func(any)) error {
		// 使用goroutine模拟异步延迟
		go func() {
			// 等待时间较长
			time.Sleep(100 * time.Millisecond)
			reject("error")
		}()
		return nil
	})
	p2 := pm.New(func(resolve, reject func(any)) error {
		// 使用goroutine模拟异步延迟
		go func() {
			// 等待时间较短
			time.Sleep(50 * time.Millisecond)
			resolve("success")
		}()
		return nil
	})

	pm.Race([]ip.Promise{p1, p2}).Then(func(v any) (any, error) {
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
	p1 := pm.New(func(resolve, reject func(any)) error {
		// 使用goroutine模拟异步延迟
		go func() {
			// 等待时间较短
			time.Sleep(50 * time.Millisecond)
			reject("error")
		}()
		return nil
	})
	p2 := pm.New(func(resolve, reject func(any)) error {
		// 使用goroutine模拟异步延迟
		go func() {
			// 等待时间较长
			time.Sleep(100 * time.Millisecond)
			resolve("success")
		}()
		return nil
	})

	pm.Race([]ip.Promise{p1, p2}).Then(func(v any) (any, error) {
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
	racePromise := pm.Race([]ip.Promise{})

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

// 测试Resolve方法 - 普通值
func TestPromiseResolve(t *testing.T) {
	pm.Resolve("value").Then(func(v any) (any, error) {
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
	original := pm.New(func(resolve, reject func(any)) error {
		resolve("original")
		return nil
	})

	p := pm.Resolve(original)

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
	pm.Reject("reason").Then(func(v any) (any, error) {
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
	pm.Try(func(args ...any) (any, error) {
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
	pm.Try(func(args ...any) (any, error) {
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
	pm.Try(nil).Then(func(v any) (any, error) {
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
	promise, resolve, _ := pm.PromiseWithResolvers()
	resolve("resolved value")

	promise.Then(func(v any) (any, error) {
		if v != "resolved value" {
			t.Errorf("Expected value 'resolved value', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})

	// 测试拒绝情况
	promise2, _, reject2 := pm.PromiseWithResolvers()
	reject2("rejected reason")

	promise2.Then(func(v any) (any, error) {
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
	initial := pm.New(func(resolve, reject func(any)) error {
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
	thenable := pm.New(func(resolve, reject func(any)) error {
		resolve("thenable value")
		return nil
	})

	p := pm.New(func(resolve, reject func(any)) error {
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
	p := pm.New(func(resolve, reject func(any)) error {
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

func TestMain(m *testing.M) {
	pm = promise.GetPromiseManager(time.Second)
	m.Run()

	// 运行事件循环以处理微任务
	pm.RunLoop(nil)
}
