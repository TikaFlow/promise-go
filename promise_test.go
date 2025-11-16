package promise_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 测试主函数
func TestMain(m *testing.M) {
	m.Run()

	<-Done()
}

// 测试Reduce方法 - 成功累加
func TestReduceSuccess(t *testing.T) {
	t.Parallel()
	p1 := Resolve(1)
	p2 := 2
	p3 := Resolve(3)

	reducer := func(acc any, item any) any {
		return acc.(int) + item.(int)
	}
	Reduce(reducer, 0, p1, p2, p3).
		Then(func(v any) (any, error) {
			if v.(int) != 6 {
				t.Errorf("Expected value 6, got %v", v.(int))
			}
			return nil, nil
		}, nil)

	<-Done()
}

// 测试Reduce方法 - 输入数组长度为0
func TestReduceEmptyArray(t *testing.T) {
	t.Parallel()
	reducer := func(acc any, item any) any {
		return acc.(int) + item.(int)
	}
	Reduce(reducer, Resolve(3), []any{}...).
		Then(func(v any) (any, error) {
			if v.(int) != 3 {
				t.Errorf("Expected value 3, got %v", v.(int))
			}
			return nil, nil
		}, nil)

	<-Done()
}

// 测试Reduce方法 - 只有一个元素且初始值为nil
func TestReduceSingleElement(t *testing.T) {
	t.Parallel()
	reducer := func(acc any, item any) any {
		return acc.(int) + item.(int)
	}
	Reduce(reducer, nil, Resolve(4)).
		Then(func(v any) (any, error) {
			if v.(int) != 4 {
				t.Errorf("Expected value 7, got %v", v.(int))
			}
			return nil, nil
		}, nil)

	<-Done()
}

// 测试Reduce方法 - 存在拒绝的Promise
func TestReduceRejected(t *testing.T) {
	t.Parallel()
	reducer := func(acc any, item any) any {
		return acc.(int) + item.(int)
	}
	Reduce(reducer, Resolve(3), Resolve(4), Reject("error")).
		Then(func(v any) (any, error) {
			t.Errorf("Promise should not be fulfilled, got value: %v", v)
			return nil, nil
		}, func(v any) (any, error) {
			if v != "error" {
				t.Errorf("Expected error 'error', got '%s'", v)
			}
			return nil, nil
		})

	<-Done()
}

// 测试Filter方法 - 全部成功
func TestFilterAllResolved(t *testing.T) {
	t.Parallel()
	p1 := 1
	p2 := Resolve(2)
	p3 := 3

	filter := func(item any) bool {
		return item.(int) > 1
	}
	Filter(filter, p1, p2, p3).
		Then(func(v any) (any, error) {
			if len(v.([]any)) != 2 {
				t.Errorf("Expected array length 2, got %d", len(v.([]any)))
			}
			if v.([]any)[0] != 2 {
				t.Errorf("Expected value 2, got %v", v.([]any)[0])
			}
			if v.([]any)[1] != 3 {
				t.Errorf("Expected value 3, got %v", v.([]any)[1])
			}
			return nil, nil
		}, nil)

	<-Done()
}

// 测试Filter方法 - 有一个拒绝
func TestFilterOneRejected(t *testing.T) {
	t.Parallel()
	p1 := Resolve(1)
	p2 := Reject("error")
	p3 := 3

	filter := func(item any) bool {
		return item.(int) > 1
	}
	Filter(filter, p1, p2, p3).
		Then(func(v any) (any, error) {
			t.Errorf("Promise should not be fulfilled, got value: %v", v)
			return nil, nil
		}, func(v any) (any, error) {
			if v != "error" {
				t.Errorf("Expected error 'error', got '%s'", v)
			}
			return nil, nil
		})

	<-Done()
}

// 测试Map方法 - 全部成功
func TestMapAllResolved(t *testing.T) {
	t.Parallel()
	p1 := Resolve(1)
	p2 := Resolve(2)
	p3 := 3
	mapper := func(item any) any {
		return item.(int) * 2
	}
	Map(mapper, p1, p2, p3).
		Then(func(v any) (any, error) {
			if len(v.([]any)) != 3 {
				t.Errorf("Expected array length 3, got %d", len(v.([]any)))
			}
			if v.([]any)[0] != 2 {
				t.Errorf("Expected value 2, got %v", v.([]any)[0])
			}
			if v.([]any)[1] != 4 {
				t.Errorf("Expected value 4, got %v", v.([]any)[1])
			}
			if v.([]any)[2] != 6 {
				t.Errorf("Expected value 6, got %v", v.([]any)[2])
			}
			return nil, nil
		}, func(v any) (any, error) {
			t.Errorf("Promise should not be rejected, got reason: %v", v)
			return nil, nil
		})

	<-Done()
}

// 测试Map方法 - 有一个拒绝
func TestMapOneRejected(t *testing.T) {
	t.Parallel()
	p1 := Resolve(1)
	p2 := Reject("error")
	p3 := 3
	mapper := func(item any) any {
		return item.(int) * 2
	}
	Map(mapper, p1, p2, p3).
		Then(func(v any) (any, error) {
			t.Errorf("Promise should not be fulfilled, got value: %v", v)
			return nil, nil
		}, func(v any) (any, error) {
			if v != "error" {
				t.Errorf("Expected error 'error', got '%s'", v)
			}
			return nil, nil
		})

	<-Done()
}

// 测试Each方法 - 所有Promise都成功
func TestEachAllSuccess(t *testing.T) {
	t.Parallel()
	p1 := Resolve(1)
	p2 := Resolve(2)
	Each(func(item any, index int, arrLen int) any {
		return item.(int) * 2
	}, p1, p2, 3).
		Then(func(v any) (any, error) {
			if len(v.([]any)) != 3 {
				t.Errorf("Expected array length 3, got %d", len(v.([]any)))
			}
			if v.([]any)[0] != 1 {
				t.Errorf("Expected value 1, got %v", v.([]any)[0])
			}
			if v.([]any)[1] != 2 {
				t.Errorf("Expected value 2, got %v", v.([]any)[1])
			}
			if v.([]any)[2] != 3 {
				t.Errorf("Expected value 3, got %v", v.([]any)[2])
			}
			return nil, nil
		}, nil)

	<-Done()
}

// 测试Each - 中间有一个失败
func TestEachOneFailure(t *testing.T) {
	t.Parallel()
	p2 := Reject("error")

	Each(func(item any, index int, arrLen int) any {
		return item.(int) * 2
	}, 1, p2, 3).
		Then(func(v any) (any, error) {
			t.Errorf("Promise should not be fulfilled, got value: %v", v)
			return nil, nil
		}, func(v any) (any, error) {
			if v != "error" {
				t.Errorf("Expected error 'error', got '%s'", v)
			}
			return nil, nil
		})

	<-Done()
}

// 测试Promise的基本创建和解决
func TestPromiseBasicResolve(t *testing.T) {
	t.Parallel()
	p := New(func(resolve, reject func(any)) error {
		resolve("success")
		return nil
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

	<-Done()
}

// 测试Promise的基本创建和拒绝
func TestPromiseBasicReject(t *testing.T) {
	t.Parallel()
	p := New(func(resolve, reject func(any)) error {
		reject("failure")
		return nil
	})

	p.Then(func(v any) (any, error) {
		t.Errorf("Promise should not be fulfilled, got value: %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "failure" {
			t.Errorf("Expected rejection reason 'failure', got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试Promise的格式化输出
func TestPromiseString(t *testing.T) {
	t.Parallel()
	Async(func() {
		p := New(func(resolve, reject func(any)) error {
			resolve("success")
			return nil
		})

		expected := "Promise<fulfilled>, result: success"
		result := fmt.Sprintf("%s", p)
		if result != expected {
			t.Errorf("Expected string '%s', got '%s'", expected, result)
		}
	})

	<-Done()
}

// 测试Promise执行器错误处理
func TestPromiseExecutorError(t *testing.T) {
	t.Parallel()
	errorMsg := "executor error"

	p := New(func(resolve, reject func(any)) error {
		return errors.New(errorMsg)
	})

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

	<-Done()
}

// 测试nil执行器
func TestPromiseNilExecutor(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for nil executor")
		}
	}()

	New(nil)
}

// 测试执行器中多次调用resolve或reject
func TestPromiseExecutorMultipleCalls(t *testing.T) {
	t.Parallel()
	slowProm := New(func(resolve, reject func(any)) error {
		SetTimeout(func() {
			resolve("slow")
		}, 200)
		return nil
	})
	fastProm := New(func(resolve, reject func(any)) error {
		SetTimeout(func() {
			resolve("fast")
		}, 100)
		return nil
	})

	p := New(func(resolve, reject func(any)) error {
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

	<-Done()
}

// 测试执行器在resolve或reject已调用后报错
func TestPromiseExecutorErrorAfterResolved(t *testing.T) {
	t.Parallel()
	delayResolve := New(func(resolve, reject func(any)) error {
		SetTimeout(func() {
			resolve("success")
		}, 100)
		return nil
	})

	p := New(func(resolve, reject func(any)) error {
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

	<-Done()
}

// 测试Then方法的基本功能 - 成功回调
func TestPromiseThenFulfilled(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
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

	<-Done()
}

// 测试Then方法的基本功能 - 拒绝回调
func TestPromiseThenRejected(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
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

	<-Done()
}

// 测试Then方法的穿透 - 成功状态
func TestPromiseThenPassThroughFulfilled(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
		resolve("value")
		return nil
	})

	p1.Then(nil, nil).Then(func(v any) (any, error) {
		if v != "value" {
			t.Errorf("Expected value 'value' after pass through, got %v", v)
		}
		return nil, nil
	}, nil)

	<-Done()
}

// 测试Then方法的穿透 - 拒绝状态
func TestPromiseThenPassThroughRejected(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
		reject("reason")
		return nil
	})

	p1.Then(nil, nil).Then(nil, func(v any) (any, error) {
		if v != "reason" {
			t.Errorf("Expected rejection reason 'reason' after pass through, got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试Then方法回调函数抛出错误
func TestPromiseThenCallbackError(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
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

	p2 := Reject("error")
	p2.Then(nil, func(v any) (any, error) {
		return "callback error", errors.New("error")
	}).Then(nil, func(v any) (any, error) {
		if v != "callback error" {
			t.Errorf("Expected error value 'callback error', got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试Then方法链式调用
func TestPromiseThenChaining(t *testing.T) {
	t.Parallel()
	New(func(resolve, reject func(any)) error {
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

	<-Done()
}

// 测试Catch方法
func TestPromiseCatch(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
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

	<-Done()
}

// 测试Catch穿透
func TestPromiseCatchPassThrough(t *testing.T) {
	t.Parallel()
	p := Resolve("success")

	p.Then(func(v any) (any, error) {
		return v.(string) + " passed 1,", nil
	}, nil).Then(func(v any) (any, error) {
		return v.(string) + " passed 2,", nil
	}, func(r any) (any, error) {
		t.Errorf("Promise should not be rejected, got reason: %v", r)
		return nil, nil
	}).Then(func(v any) (any, error) {
		return v.(string) + " rejected", errors.New("error")
	}, nil).Then(func(v any) (any, error) {
		t.Errorf("Promise should not be fulfilled, got value: %v", v)
		return nil, nil
	}, nil).Catch(func(r any) (any, error) {
		if r != "success passed 1, passed 2, rejected" {
			t.Errorf("Expected value 'success passed 1, passed 2, rejected', got %v", r)
		}
		return nil, nil
	})

	<-Done()
}

// 测试Finally方法 - 成功状态
func TestPromiseFinallyFulfilled(t *testing.T) {
	t.Parallel()
	finallyCalled := false

	p1 := New(func(resolve, reject func(any)) error {
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

	<-Done()
}

// 测试Finally方法 - 拒绝状态
func TestPromiseFinallyRejected(t *testing.T) {
	t.Parallel()
	finallyCalled := false

	p1 := New(func(resolve, reject func(any)) error {
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

	<-Done()
}

// 测试Finally方法抛出错误
func TestPromiseFinallyError(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
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

	<-Done()
}

// 测试Finally方法返回被拒绝的Promise
func TestPromiseFinallyReturnsRejectedPromise(t *testing.T) {
	t.Parallel()
	rejectedPromise := New(func(resolve, reject func(any)) error {
		reject("rejected from finally")
		return nil
	})

	p1 := New(func(resolve, reject func(any)) error {
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

	<-Done()
}

// 测试All方法 - 所有Promise都成功
func TestPromiseAllFulfilled(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
		resolve(1)
		return nil
	})
	p2 := New(func(resolve, reject func(any)) error {
		resolve(2)
		return nil
	})
	p3 := New(func(resolve, reject func(any)) error {
		resolve(3)
		return nil
	})

	All(p1, p2, p3).Then(func(v any) (any, error) {
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

	<-Done()
}

// 测试All方法 - 含有非Promise对象
func TestPromiseAllNotPromise(t *testing.T) {
	t.Parallel()
	p1 := "string"
	p2 := 2
	p3 := false

	All(p1, p2, p3).Then(func(v any) (any, error) {
		results, ok := v.([]any)
		if !ok {
			t.Errorf("Expected []any type, got %T", v)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}
		if results[0] != p1 || results[1] != p2 || results[2] != p3 {
			t.Errorf("Expected [%v, %v, %v], got %v", p1, p2, p3, results)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.All should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})

	<-Done()
}

// 测试All方法 - 有一个Promise被拒绝
func TestPromiseAllRejected(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
		resolve(1)
		return nil
	})
	p2 := New(func(resolve, reject func(any)) error {
		reject("error")
		return nil
	})
	p3 := New(func(resolve, reject func(any)) error {
		resolve(3)
		return nil
	})

	All(p1, p2, p3).Then(func(v any) (any, error) {
		t.Errorf("Promise.All should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "error" {
			t.Errorf("Expected rejection value 'error', got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试All方法 - 空数组
func TestPromiseAllEmptyArray(t *testing.T) {
	t.Parallel()
	allPromise := All(make([]any, 0)...)

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

	<-Done()
}

// 测试All方法 - nil数组
func TestPromiseAllNilArray(t *testing.T) {
	t.Parallel()
	All().Then(func(v any) (any, error) {
		t.Errorf("Promise.All with nil array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "TypeError: nil is not iterable" {
			t.Errorf("Expected error value 'TypeError: nil is not iterable', got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试AllSettled方法
func TestPromiseAllSettled(t *testing.T) {
	t.Parallel()

	p1 := New(func(resolve, reject func(any)) error {
		resolve(1)
		return nil
	})
	p2 := New(func(resolve, reject func(any)) error {
		reject("error")
		return nil
	})
	p3 := New(func(resolve, reject func(any)) error {
		resolve(3)
		return nil
	})

	AllSettled(p1, p2, p3).Then(func(v any) (any, error) {
		results, ok := v.([]map[string]any)
		if !ok {
			t.Errorf("Expected []map[string]any type, got %T", v)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		if results[0]["status"] != Fulfilled || results[0]["value"] != 1 {
			t.Errorf("Expected first result to be fulfilled with value 1, got %v", results[0])
		}
		if results[1]["status"] != Rejected || results[1]["reason"] != "error" {
			t.Errorf("Expected second result to be rejected with reason 'error', got %v", results[1])
		}
		if results[2]["status"] != Fulfilled || results[2]["value"] != 3 {
			t.Errorf("Expected third result to be fulfilled with value 3, got %v", results[2])
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.AllSettled should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})

	<-Done()
}

// 测试AllSettled方法 - 空数组
func TestPromiseAllSettledEmptyArray(t *testing.T) {
	t.Parallel()
	AllSettled(make([]any, 0)...).Then(func(v any) (any, error) {
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

	<-Done()
}

// 测试AllSettled方法 - nil数组
func TestPromiseAllSettledNilArray(t *testing.T) {
	t.Parallel()
	AllSettled().Then(func(v any) (any, error) {
		t.Errorf("Promise.AllSettled with nil array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "TypeError: nil is not iterable" {
			t.Errorf("Expected error value 'TypeError: nil is not iterable', got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试Any方法 - 有一个Promise成功
func TestPromiseAnyFulfilled(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
		reject("error1")
		return nil
	})
	p2 := New(func(resolve, reject func(any)) error {
		resolve("success")
		return nil
	})
	p3 := New(func(resolve, reject func(any)) error {
		reject("error2")
		return nil
	})

	Any(p1, p2, p3).Then(func(v any) (any, error) {
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.Any should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})

	<-Done()
}

// 测试Any方法 - 所有Promise都失败
func TestPromiseAnyAllRejected(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
		reject("error1")
		return nil
	})
	p2 := New(func(resolve, reject func(any)) error {
		reject("error2")
		return nil
	})

	Any(p1, p2).Then(func(v any) (any, error) {
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

	<-Done()
}

// 测试Any方法 - 空数组
func TestPromiseAnyEmptyArray(t *testing.T) {
	t.Parallel()
	Any(make([]any, 0)...).Then(func(v any) (any, error) {
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

	<-Done()
}

// 测试Any方法 - nil数组
func TestPromiseAnyNilArray(t *testing.T) {
	t.Parallel()
	Any().Then(func(v any) (any, error) {
		t.Errorf("Promise.Any with nil array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "TypeError: nil is not iterable" {
			t.Errorf("Expected error value 'TypeError: nil is not iterable', got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试Some方法 - num<=0
func TestPromiseSomeNumLE0(t *testing.T) {
	t.Parallel()
	Some(0, Resolve("success")).Then(func(v any) (any, error) {
		t.Errorf("Promise.Some with num<=0 should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "RangeError: num must be greater than 0" {
			t.Errorf("Expected error value 'RangeError: num must be greater than 0', got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试Some方法 - num>proms长度
func TestPromiseSomeNumGTPromsLen(t *testing.T) {
	t.Parallel()
	Some(2, Resolve("success")).Then(func(v any) (any, error) {
		t.Errorf("Promise.Some with num>proms length should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "RangeError: not enough promises to resolve" {
			t.Errorf("Expected error value 'RangeError: not enough promises to resolve', got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试Some方法 - 3个满足2个
func TestPromiseSome2in3(t *testing.T) {
	t.Parallel()
	p1 := Resolve("success1")
	p2 := Resolve("success2")
	p3 := Reject("failure")

	Some(2, p1, p2, p3).Then(func(v any) (any, error) {
		if len(v.([]any)) != 2 {
			t.Errorf("Expected 2 values, got %d", len(v.([]any)))
			return nil, nil
		}
		if v.([]any)[0] != "success1" || v.([]any)[1] != "success2" {
			t.Errorf("Expected values ['success1','success2'], got %v", v.([]any))
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.Some should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})

	<-Done()
}

// 测试Some方法 - 3个拒绝2个
func TestPromiseSome2out3(t *testing.T) {
	t.Parallel()
	p1 := Reject("failure1")
	p2 := Reject("failure2")
	p3 := Resolve("success")

	Some(2, p1, p2, p3).Then(func(v any) (any, error) {
		t.Errorf("Promise.Some should be rejected, but was fulfilled with %v", v)
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
		if agg["errors"].([]any)[0] != "failure1" || agg["errors"].([]any)[1] != "failure2" {
			t.Errorf("Expected errors ['failure1','failure2'], got %v", agg["errors"].([]any))
		}
		return nil, nil
	})

	<-Done()
}

// 测试Race方法 - 第一个完成的是成功的Promise
func TestPromiseRaceFulfilled(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
		SetTimeout(func() {
			reject("error")
		}, 100)
		return nil
	})
	p2 := New(func(resolve, reject func(any)) error {
		SetTimeout(func() {
			resolve("success")
		}, 50)
		return nil
	})

	Race(p1, p2).Then(func(v any) (any, error) {
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.Race should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})

	<-Done()
}

// 测试Race方法 - 第一个完成的是失败的Promise
func TestPromiseRaceRejected(t *testing.T) {
	t.Parallel()
	p1 := New(func(resolve, reject func(any)) error {
		SetTimeout(func() {
			reject("error")
		}, 50)
		return nil
	})
	p2 := New(func(resolve, reject func(any)) error {
		SetTimeout(func() {
			resolve("success")
		}, 100)
		return nil
	})

	Race(p1, p2).Then(func(v any) (any, error) {
		t.Errorf("Promise.Race should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "error" {
			t.Errorf("Expected value 'error', got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试Race方法 - 空数组
func TestPromiseRaceEmptyArray(t *testing.T) {
	t.Parallel()
	racePromise := Race(make([]any, 0)...)
	var state string

	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-racePromise.Done():
		state = racePromise.State()
	case <-timer.C:
		state = Pending
	}

	if state != Pending {
		t.Errorf("Expected state Pending for empty array Race, got %s", state)
	}

	<-Done()
}

// 测试Race方法 - nil数组
func TestPromiseRaceNilArray(t *testing.T) {
	t.Parallel()
	Race().Then(func(v any) (any, error) {
		t.Errorf("Promise.Race with nil array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "TypeError: nil is not iterable" {
			t.Errorf("Expected error value 'TypeError: nil is not iterable', got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试Resolve方法 - 普通值
func TestPromiseResolve(t *testing.T) {
	t.Parallel()
	Resolve("value").Then(func(v any) (any, error) {
		if v != "value" {
			t.Errorf("Expected value 'value', got %v", v)
		}
		return nil, nil
	}, func(v any) (any, error) {
		t.Errorf("Promise.Resolve should be fulfilled, but was rejected with %v", v)
		return nil, nil
	})

	<-Done()
}

// 测试Resolve方法 - Promise对象 - fulfilled状态
func TestPromiseResolvePromise(t *testing.T) {
	t.Parallel()
	original := New(func(resolve, reject func(any)) error {
		resolve("original")
		return nil
	})

	p := Resolve(original)

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

	<-Done()
}

// 测试Resolve方法 - Promise对象 - rejected状态
func TestPromiseResolvePromiseRejected(t *testing.T) {
	t.Parallel()
	original := New(func(resolve, reject func(any)) error {
		reject("error")
		return nil
	})

	p := Resolve(original)

	if p != original {
		t.Errorf("Expected Resolve to return the same Promise instance")
	}

	p.Then(func(v any) (any, error) {
		t.Errorf("Expected state Rejected, got Fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "error" {
			t.Errorf("Expected value 'error', got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试Reject方法
func TestPromiseReject(t *testing.T) {
	t.Parallel()
	Reject("reason").Then(func(v any) (any, error) {
		t.Errorf("Promise.Reject should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v any) (any, error) {
		if v != "reason" {
			t.Errorf("Expected value 'reason', got %v", v)
		}
		return nil, nil
	})

	<-Done()
}

// 测试Try方法 - 成功
func TestPromiseTrySuccess(t *testing.T) {
	t.Parallel()
	Try(func(args ...any) (any, error) {
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

	<-Done()
}

// 测试Try方法 - 失败
func TestPromiseTryError(t *testing.T) {
	t.Parallel()
	Try(func(args ...any) (any, error) {
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

	<-Done()
}

// 测试Try方法 - nil函数
func TestPromiseTryNilFunc(t *testing.T) {
	t.Parallel()
	Try(nil).Then(func(v any) (any, error) {
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

	<-Done()
}

// 测试PromiseWithResolvers方法
func TestPromiseWithResolvers(t *testing.T) {
	t.Parallel()
	// 测试成功情况
	p, resolve, _ := PromiseWithResolvers()
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
	p2, _, reject2 := PromiseWithResolvers()
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

	<-Done()
}

// 测试循环引用检测
func TestPromiseCycleDetection(t *testing.T) {
	t.Parallel()
	initial := New(func(resolve, reject func(any)) error {
		resolve("initial")
		return nil
	})

	var p Promise
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

	<-Done()
}

// 测试Thenable对象处理
func TestPromiseThenable(t *testing.T) {
	t.Parallel()
	result := "init"

	Async(func() {
		p1, resolveP1, _ := PromiseWithResolvers()
		p2 := New(func(resolve, reject func(any)) error {
			resolve(p1)
			return nil
		})
		resolveP1("thenable value")

		p1.Then(func(v any) (any, error) {
			result += " =>p1-resolved"
			QueueMicrotask(func() {
				result += " =>p1:microtask"
			})
			return nil, nil
		}, nil)

		p2.Then(func(v any) (any, error) {
			result += " =>p2<" + v.(string) + ">"
			QueueMicrotask(func() {
				result += " =>p2:microtask"
			})
			return nil, nil
		}, nil)

		QueueMicrotask(func() {
			result += " =>microtask"
		})
	})

	SetTimeout(func() {
		expected := "init =>p1-resolved =>microtask =>p1:microtask =>p2<thenable value> =>p2:microtask"
		if result != expected {
			t.Errorf("Expected result '\n%s', got '\n%s'", expected, result)
		}
	}, 10)

	<-Done()
}

// 测试多个Then调用的顺序
func TestPromiseMultipleThenOrder(t *testing.T) {
	t.Parallel()
	p := New(func(resolve, reject func(any)) error {
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

	<-Done()
}

// 测试SetTimeout函数
func TestSetTimeout(t *testing.T) {
	t.Parallel()
	var str string
	SetTimeout(func() {
		str = "timeout value"
	}, 100)

	SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 80)

	SetTimeout(func() {
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	}, 120)

	<-Done()
}

// 测试SetTimeout函数 - 取消
func TestSetTimeoutCancel(t *testing.T) {
	t.Parallel()
	p := New(func(resolve, reject func(any)) error {
		id := SetTimeout(func() {
			resolve("timeout value")
		}, 100)
		ClearTimeout(id)
		return nil
	})

	SetTimeout(func() {
		if p.State() != Pending {
			t.Errorf("Expected state Pending, got %v", p.State())
		}
	}, 120)

	<-Done()
}

// 测试SetTimeout函数 - 毫秒数为负数
func TestSetTimeoutNegativeMillis(t *testing.T) {
	t.Parallel()
	var str string
	SetTimeout(func() {
		str = "timeout value"
	}, -100)

	SetTimeout(func() {
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	}, 20)

	<-Done()
}

// 测试SetTimeout函数 - 毫秒数为 0
func TestSetTimeoutZeroMillis(t *testing.T) {
	t.Parallel()
	var str string
	SetTimeout(func() {
		str = "timeout value"
	}, 0)

	Async(func() {
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	})

	<-Done()
}

// 测试SetTimeout函数 - 长延迟
func TestSetTimeoutLongDelay(t *testing.T) {
	t.Parallel()
	var str string
	SetTimeout(func() {
		str = "timeout value"
	}, 1000)

	SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 980)

	SetTimeout(func() {
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	}, 1020)

	<-Done()
}

// 测试SetInterval函数
func TestSetInterval(t *testing.T) {
	t.Parallel()
	var str string
	var count int
	var id int

	id = SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			ClearInterval(id)
		}
	}, 200)

	SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 180)

	SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 220)

	SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 380)

	SetTimeout(func() {
		if str != "interval interval " {
			t.Errorf("Expected str 'interval interval ', got %s", str)
		}
	}, 420)

	SetTimeout(func() {
		if str != "interval interval " {
			t.Errorf("Expected str 'interval interval ', got %s", str)
		}
	}, 580)

	SetTimeout(func() {
		if str != "interval interval interval " {
			t.Errorf("Expected str 'interval interval interval ', got %s", str)
		}
	}, 620)

	<-Done()
}

// 测试SetInterval函数 - 取消 - 第1次执行
func TestSetIntervalCancelFirst(t *testing.T) {
	t.Parallel()
	var str string
	var count int
	var id int

	id = SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			ClearInterval(id)
		}
	}, 200)

	SetTimeout(func() {
		ClearInterval(id)
	}, 20)

	SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 240)

	<-Done()
}

// 测试SetInterval函数 - 取消 - 非首次执行
func TestSetIntervalCancelNonFirst(t *testing.T) {
	t.Parallel()
	var str string
	var count int
	var id int

	id = SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			ClearInterval(id)
		}
	}, 200)

	SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
		ClearInterval(id)
	}, 220)

	SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 620)

	<-Done()
}

// 测试SetInterval函数 - 长延迟
func TestSetIntervalLongDelay(t *testing.T) {
	t.Parallel()
	var str string
	var id int

	count := 0
	id = SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			ClearInterval(id)
		}
	}, 1000)

	SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 980)
	SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 1020)

	SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 1980)
	SetTimeout(func() {
		if str != "interval interval " {
			t.Errorf("Expected str 'interval interval ', got %s", str)
		}
	}, 2020)

	SetTimeout(func() {
		if str != "interval interval " {
			t.Errorf("Expected str 'interval interval ', got %s", str)
		}
	}, 2980)
	SetTimeout(func() {
		if str != "interval interval interval " {
			t.Errorf("Expected str 'interval interval interval ', got %s", str)
		}
	}, 3020)

	<-Done()
}

// 测试Await - 成功
func TestAwaitSuccess(t *testing.T) {
	t.Parallel()
	p := New(func(resolve, reject func(any)) error {
		resolve("success")
		return nil
	})

	res, err := Await(p, 50)
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if res != "success" {
		t.Errorf("Expected 'success', got %s", res)
	}
}

// 测试Await - timeout不是正数
func TestAwaitTimeoutNotPositive(t *testing.T) {
	t.Parallel()
	p := New(func(resolve, reject func(any)) error {
		resolve("success")
		return nil
	})

	expected := "await timeout must be greater than 0"
	res, err := Await(p, -100)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res.(error).Error() != expected {
		t.Errorf("Expected error '%s', got %s", expected, res.(error).Error())
	}

	res, err = Await(p, -0)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res.(error).Error() != expected {
		t.Errorf("Expected error '%s', got %s", expected, res.(error).Error())
	}
}

// 测试Await - 超时
func TestAwaitTimeout(t *testing.T) {
	t.Parallel()
	p := New(func(resolve, reject func(any)) error {
		SetTimeout(func() {
			resolve("success")
		}, 100)
		return nil
	})

	res, err := Await(p, 50)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res.(error).Error() != "TimeoutError: await timeout" {
		t.Errorf("Expected error 'TimeoutError: await timeout', got %s", res.(error).Error())
	}
}

// 测试Await - 拒绝的Promise
func TestAwaitRejectedPromise(t *testing.T) {
	t.Parallel()
	p := New(func(resolve, reject func(any)) error {
		reject("error")
		return nil
	})

	res, err := Await(p, 50)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != "error" {
		t.Errorf("Expected error 'error', got %s", res)
	}
}

// 测试Delay函数
func TestDelay(t *testing.T) {
	t.Parallel()
	val := "value"

	if _, err := Await(Delay(val, 50), 40); err == nil {
		// timeout
		t.Errorf("Expected 'TimeoutError: await timeout', got nil")
	}

	res, err := Await(Delay(val, 50), 60)
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if res != val {
		t.Errorf("Expected '%s', got %s", val, res)
	}

	p := New(func(resolve, reject func(any)) error {
		SetTimeout(func() {
			resolve("success")
		}, 50)
		return nil
	})
	if _, err = Await(Delay(p, 50), 90); err == nil {
		// timeout
		t.Errorf("Expected 'TimeoutError: await timeout', got nil")
	}

	p2 := New(func(resolve, reject func(any)) error {
		SetTimeout(func() {
			resolve("success")
		}, 50)
		return nil
	})
	if res, _ := Await(Delay(p2, 50), 110); res != "success" {
		t.Errorf("Expected 'success', got %s", res)
	}
}

// 测试异步调用顺序 - 微任务
func TestAsyncCallOrderMicro(t *testing.T) {
	t.Parallel()
	result := ""
	Async(func() {
		var res func(any)
		p := Resolve("success")
		p.Then(func(v any) (any, error) {
			result += "[B]"
			return nil, nil
		}, nil).Then(func(v any) (any, error) {
			result += "[D]"
			return nil, nil
		}, nil)
		New(func(resolve, reject func(any)) error {
			result += "[A]"
			res = resolve
			return nil
		}).Then(func(v any) (any, error) {
			result += "[E]"
			return nil, nil
		}, nil)
		res(p)
		QueueMicrotask(func() {
			result += "[C]"
		})
	})

	Async(func() {
		if result != "[A][B][C][D][E]" {
			t.Errorf("Expected result1 '[A][B][C][D][E]', got %s", result)
		}
	})

	<-Done()
}

// 测试异步调用顺序 - 宏任务
func TestAsyncCallOrderMacro(t *testing.T) {
	t.Parallel()
	result := ""
	SetTimeout(func() {
		result += "[A]"
		SetTimeout(func() {
			result += "[C]"
		}, 20)
	}, 30)
	// 无延迟
	SetTimeout(func() {
		result += "[B]"
	}, 50)

	SetTimeout(func() {
		if result != "[A][B][C]" {
			t.Errorf("Expected result1 '[A][B][C]', got %s", result)
		}
	}, 60)

	<-Done()
}

// 测试异步调用顺序 - 宏任务 - 有延迟
func TestAsyncCallOrderMacroDelay(t *testing.T) {
	t.Parallel()
	result := ""

	SetTimeout(func() {
		result += "[A]"
		SetTimeout(func() {
			result += "[C]"
		}, 20)
	}, 30)
	// 有延迟 - 看情况调大数字，go实在太快了，区区循环嗖一下就完成了
	for i := range 12345678 {
		_ = int64(i) * 1234 / 2234 % 3234
	}
	SetTimeout(func() {
		result += "[B]"
	}, 50)

	SetTimeout(func() {
		if result != "[A][C][B]" {
			t.Errorf("Expected result2 '[A][C][B]', got %s", result)
		}
	}, 60)

	<-Done()
}

// 测试异步调用顺序 - 混合模式
func TestAsyncCallOrderMixed(t *testing.T) {
	t.Parallel()
	var result string

	Async(func() {
		result += "[01]"

		SetTimeout(func() {
			result += "-[28]"

			Reject("error").Catch(func(v any) (any, error) {
				result += "-[29]"
				return nil, nil
			})

			QueueMicrotask(func() {
				result += "-[30]"
				/*
				   根据ES规范，37应该出现在40后面，但在浏览器中，由于0ms小于最低延迟时间（通常是4ms），
				   导致37出现在36和38之间，即浏览器执行是01-42完全连续（浏览器对0ms延迟做了特别处理）。
				   注：37和31、35、36、38、39的理论触发时间相同，根据ES规范，先注册先执行，37是最后注册的，
				   但由于执行延迟（见 [TestAsyncCallOrderMacroDelay]），37可能出现在上述5个任务的任意一个之前，是正常现象。
				*/
				SetTimeout(func() {
					result += "-[37]"
				}, 0)
			})

			SetTimeout(func() {
				result += "-[41]"
			}, 30)
		}, 50)

		p1 := Resolve(nil)
		p1.Then(func(v any) (any, error) {
			result += "-[04]"

			Resolve(nil).Then(func(v any) (any, error) {
				result += "-[10]"
				return nil, nil
			}, nil)

			QueueMicrotask(func() {
				result += "-[11]"
				SetTimeout(func() {
					result += "-[38]"
				}, 50)
			})

			SetTimeout(func() {
				result += "-[26]"
			}, 0)

			return nil, nil
		}, nil)

		p2 := Resolve(nil)
		p2.Then(func(v any) (any, error) {
			result += "-[05]"
			return nil, nil
		}, nil).Then(func(v any) (any, error) {
			result += "-[12]"

			Resolve(nil).Then(func(v any) (any, error) {
				result += "-[19]"
				return nil, nil
			}, nil).Then(func(v any) (any, error) {
				result += "-[24]"
				return nil, nil
			}, nil)

			QueueMicrotask(func() {
				result += "-[20]"
			})

			SetTimeout(func() {
				result += "-[39]"

				QueueMicrotask(func() {
					result += "-[40]"
				})
			}, 50)

			return nil, nil
		}, nil).Then(func(v any) (any, error) {
			result += "-[21]"
			return nil, nil
		}, nil)

		p3 := New(func(resolve, reject func(any)) error {
			result += "-[02]"
			resolve(4)
			return nil
		})

		p3.Then(func(v any) (any, error) {
			result += "-[06]"

			Reject("error").Catch(func(v any) (any, error) {
				result += "-[13]"

				Resolve(nil).Then(func(v any) (any, error) {
					result += "-[22]"
					return nil, nil
				}, nil)

				return nil, nil
			})

			QueueMicrotask(func() {
				result += "-[14]"
			})

			SetTimeout(func() {
				result += "-[35]"
			}, 50)

			return nil, nil
		}, nil)

		p3.Then(func(v any) (any, error) {
			result += "-[07]"

			Resolve(nil).Finally(func() (any, error) {
				result += "-[15]"
				return nil, nil
			})

			QueueMicrotask(func() {
				result += "-[16]"
			})

			SetTimeout(func() {
				result += "-[36]"
			}, 50)

			return nil, nil
		}, nil)

		p3.Then(func(v any) (any, error) {
			result += "-[08]"
			return nil, nil
		}, nil)

		count := 1
		var id int
		id = SetInterval(func() {
			result += "-[31]"

			if count >= 2 {
				ClearInterval(id)
			}
			count++

			Resolve(nil).Finally(func() (any, error) {
				result += "-[32]"

				Resolve(nil).Finally(func() (any, error) {
					result += "-[34]"
					return nil, nil
				})

				return nil, nil
			})

			QueueMicrotask(func() {
				result += "-[33]"
			})

			SetTimeout(func() {
				result += "-[42]"
			}, 30)
		}, 50)

		QueueMicrotask(func() {
			result += "-[09]"

			p := Resolve(nil)
			p.Then(func(v any) (any, error) {
				result += "-[17]"
				return nil, nil
			}, nil).Then(func(v any) (any, error) {
				result += "-[23]"
				return nil, nil
			}, nil).Then(func(v any) (any, error) {
				result += "-[25]"
				return nil, nil
			}, nil)

			QueueMicrotask(func() {
				result += "-[18]"
			})

			SetTimeout(func() {
				result += "-[27]"
			}, 0)
		})

		result += "-[03]"
	})

	SetTimeout(func() {

		expected := "[01]-[02]-[03]-[04]-[05]-[06]-[07]-[08]-[09]-[10]-[11]-[12]-[13]-[14]-[15]-[16]-[17]-[18]-[19]-[20]-[21]-[22]-[23]-[24]-[25]-[26]-[27]-[28]-[29]-[30]-[31]-[32]-[33]-[34]-[35]-[36]-[38]-[39]-[40]-[37]-[41]-[42]-[31]-[32]-[33]-[34]-[42]"
		if result != expected {
			t.Errorf("Expected result:\n%s\n\nGot:\n%s", expected, result)
		}
	}, 360)

	<-Done()
}
