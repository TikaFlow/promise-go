package promise_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

var el EventLoop

// 测试主函数
func TestMain(m *testing.M) {
	el = StartClassicEventLoop()
	m.Run()

	_ = el.Close()
}

// 测试Reduce方法 - 成功累加
func TestReduceSuccess(t *testing.T) {
	t.Parallel()
	p1 := el.Resolve(1)
	p2 := 2
	p3 := el.Resolve(3)

	reducer := func(acc any, item any) any {
		return acc.(int) + item.(int)
	}
	el.Reduce(reducer, 0, p1, p2, p3).
		Then(func(v any) (any, error) {
			if v.(int) != 6 {
				t.Errorf("Expected value 6, got %v", v.(int))
			}
			return nil, nil
		}, nil)
}

// 测试Reduce方法 - 输入数组长度为0
func TestReduceEmptyArray(t *testing.T) {
	t.Parallel()
	reducer := func(acc any, item any) any {
		return acc.(int) + item.(int)
	}
	el.Reduce(reducer, el.Resolve(3), []any{}...).
		Then(func(v any) (any, error) {
			if v.(int) != 3 {
				t.Errorf("Expected value 3, got %v", v.(int))
			}
			return nil, nil
		}, nil)
}

// 测试Reduce方法 - 只有一个元素且初始值为nil
func TestReduceSingleElement(t *testing.T) {
	t.Parallel()
	reducer := func(acc any, item any) any {
		return acc.(int) + item.(int)
	}
	el.Reduce(reducer, nil, el.Resolve(4)).
		Then(func(v any) (any, error) {
			if v.(int) != 4 {
				t.Errorf("Expected value 7, got %v", v.(int))
			}
			return nil, nil
		}, nil)
}

// 测试Reduce方法 - 存在拒绝的Promise
func TestReduceRejected(t *testing.T) {
	t.Parallel()
	reducer := func(acc any, item any) any {
		return acc.(int) + item.(int)
	}
	el.Reduce(reducer, el.Resolve(3), el.Resolve(4), el.Reject("error")).
		Then(func(v any) (any, error) {
			t.Errorf("Promise should not be fulfilled, got value: %v", v)
			return nil, nil
		}, func(v error) (any, error) {
			if v.Error() != "UnexpectedError: error" {
				t.Errorf("Expected error 'UnexpectedError: error', got '%s'", v.Error())
			}
			return nil, nil
		})
}

// 测试Filter方法 - 全部成功
func TestFilterAllResolved(t *testing.T) {
	t.Parallel()
	p1 := 1
	p2 := el.Resolve(2)
	p3 := 3

	filter := func(item any) bool {
		return item.(int) > 1
	}
	el.Filter(filter, p1, p2, p3).
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
}

// 测试Filter方法 - 有一个拒绝
func TestFilterOneRejected(t *testing.T) {
	t.Parallel()
	p1 := el.Resolve(1)
	p2 := el.Reject("error")
	p3 := 3

	filter := func(item any) bool {
		return item.(int) > 1
	}
	el.Filter(filter, p1, p2, p3).
		Then(func(v any) (any, error) {
			t.Errorf("Promise should not be fulfilled, got value: %v", v)
			return nil, nil
		}, func(v error) (any, error) {
			if v.Error() != "UnexpectedError: error" {
				t.Errorf("Expected error 'UnexpectedError: error', got '%s'", v.Error())
			}
			return nil, nil
		})
}

// 测试Map方法 - 全部成功
func TestMapAllResolved(t *testing.T) {
	t.Parallel()
	p1 := el.Resolve(1)
	p2 := el.Resolve(2)
	p3 := 3
	mapper := func(item any) any {
		return item.(int) * 2
	}
	el.Map(mapper, p1, p2, p3).
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
		}, func(v error) (any, error) {
			t.Errorf("Promise should not be rejected, got reason: %v", v.Error())
			return nil, nil
		})
}

// 测试Map方法 - 有一个拒绝
func TestMapOneRejected(t *testing.T) {
	t.Parallel()
	p1 := el.Resolve(1)
	p2 := el.Reject("error")
	p3 := 3
	mapper := func(item any) any {
		return item.(int) * 2
	}
	el.Map(mapper, p1, p2, p3).
		Then(func(v any) (any, error) {
			t.Errorf("Promise should not be fulfilled, got value: %v", v)
			return nil, nil
		}, func(v error) (any, error) {
			if v.Error() != "UnexpectedError: error" {
				t.Errorf("Expected error 'UnexpectedError: error', got '%s'", v.Error())
			}
			return nil, nil
		})
}

// 测试Each方法 - 所有Promise都成功
func TestEachAllSuccess(t *testing.T) {
	t.Parallel()
	p1 := el.Resolve(1)
	p2 := el.Resolve(2)
	el.Each(func(item any, index int, arrLen int) any {
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
}

// 测试Each - 中间有一个失败
func TestEachOneFailure(t *testing.T) {
	t.Parallel()
	p2 := el.Reject("error")

	el.Each(func(item any, index int, arrLen int) any {
		return item.(int) * 2
	}, 1, p2, 3).
		Then(func(v any) (any, error) {
			t.Errorf("Promise should not be fulfilled, got value: %v", v)
			return nil, nil
		}, func(v error) (any, error) {
			if v.Error() != "UnexpectedError: error" {
				t.Errorf("Expected error 'UnexpectedError: error', got '%s'", v.Error())
			}
			return nil, nil
		})
}

// 测试Promise的基本创建和解决
func TestPromiseBasicResolve(t *testing.T) {
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

// 测试Promise的基本创建和拒绝
func TestPromiseBasicReject(t *testing.T) {
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

// 测试Promise的格式化输出
func TestPromiseString(t *testing.T) {
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
}

// 测试Promise执行器错误处理
func TestPromiseExecutorError(t *testing.T) {
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

// 测试nil执行器
func TestPromiseNilExecutor(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for nil executor")
		}
	}()

	el.NewPromise(nil)
}

// 测试执行器中多次调用resolve或reject
func TestPromiseExecutorMultipleCalls(t *testing.T) {
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
}

// 测试执行器在resolve或reject已调用后报错
func TestPromiseExecutorErrorAfterResolved(t *testing.T) {
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
}

// 测试Then方法的基本功能 - 成功回调
func TestPromiseThenFulfilled(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
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
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})

	p1.Then(nil, func(v error) (any, error) {
		return "handled: " + v.Error(), nil
	}).Then(func(v any) (any, error) {
		if v != "handled: UnexpectedError: error" {
			t.Errorf("Expected value 'handled: UnexpectedError: error', got %v", v)
		}
		return nil, nil
	}, nil)
}

// 测试Then方法的穿透 - 成功状态
func TestPromiseThenPassThroughFulfilled(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
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
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("reason")
		return nil
	})

	p1.Then(nil, nil).Then(nil, func(v error) (any, error) {
		if v.Error() != "UnexpectedError: reason" {
			t.Errorf("Expected rejection reason 'UnexpectedError: reason' after pass through, got %v", v.Error())
		}
		return nil, nil
	})
}

// 测试Then方法回调函数抛出错误
func TestPromiseThenCallbackError(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("value")
		return nil
	})

	p1.Then(func(v any) (any, error) {
		return nil, errors.New("callback error")
	}, nil).Then(nil, func(v error) (any, error) {
		if v.Error() != "callback error" {
			t.Errorf("Expected error value 'callback error', got %v", v.Error())
		}
		return nil, nil
	})

	p2 := el.Reject("error")
	p2.Then(nil, func(v error) (any, error) {
		return nil, errors.New("callback error")
	}).Then(nil, func(v error) (any, error) {
		if v.Error() != "callback error" {
			t.Errorf("Expected error value 'callback error', got %v", v.Error())
		}
		return nil, nil
	})
}

// 测试Then方法链式调用
func TestPromiseThenChaining(t *testing.T) {
	t.Parallel()
	el.NewPromise(func(resolve, reject func(v any)) error {
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
func TestPromiseCatchPassThrough(t *testing.T) {
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

// 测试Finally方法 - 成功状态
func TestPromiseFinallyFulfilled(t *testing.T) {
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
func TestPromiseFinallyRejected(t *testing.T) {
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

// 测试Finally方法抛出错误
func TestPromiseFinallyError(t *testing.T) {
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

// 测试Finally方法返回被拒绝的Promise
func TestPromiseFinallyReturnsRejectedPromise(t *testing.T) {
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

// 测试All方法 - 所有Promise都成功
func TestPromiseAllFulfilled(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve(1)
		return nil
	})
	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve(2)
		return nil
	})
	p3 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve(3)
		return nil
	})

	el.All(p1, p2, p3).Then(func(v any) (any, error) {
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
	}, func(v error) (any, error) {
		t.Errorf("Promise.All should be fulfilled, but was rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试All方法 - 含有非Promise对象
func TestPromiseAllNotPromise(t *testing.T) {
	t.Parallel()
	p1 := "string"
	p2 := 2
	p3 := false

	el.All(p1, p2, p3).Then(func(v any) (any, error) {
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
	}, func(v error) (any, error) {
		t.Errorf("Promise.All should be fulfilled, but was rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试All方法 - 有一个Promise被拒绝
func TestPromiseAllRejected(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve(1)
		return nil
	})
	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})
	p3 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve(3)
		return nil
	})

	el.All(p1, p2, p3).Then(func(v any) (any, error) {
		t.Errorf("Promise.All should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "UnexpectedError: error" {
			t.Errorf("Expected rejection value 'UnexpectedError: error', got %s", v.Error())
		}
		return nil, nil
	})
}

// 测试All方法 - 空数组
func TestPromiseAllEmptyArray(t *testing.T) {
	t.Parallel()
	allPromise := el.All(make([]any, 0)...)

	allPromise.Then(func(v any) (any, error) {
		results, ok := v.([]any)
		if !ok {
			t.Errorf("Expected []any type, got %T", v)
		}
		if len(results) != 0 {
			t.Errorf("Expected empty array, got %d elements", len(results))
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise.All should be fulfilled, but was rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试All方法 - nil数组
func TestPromiseAllNilArray(t *testing.T) {
	t.Parallel()
	el.All().Then(func(v any) (any, error) {
		t.Errorf("Promise.All with nil array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "TypeError: nil is not iterable" {
			t.Errorf("Expected error value 'TypeError: nil is not iterable', got %v", v.Error())
		}
		return nil, nil
	})
}

// 测试AllSettled方法
func TestPromiseAllSettled(t *testing.T) {
	t.Parallel()

	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve(1)
		return nil
	})
	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})
	p3 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve(3)
		return nil
	})

	el.AllSettled(p1, p2, p3).Then(func(v any) (any, error) {
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
		if results[1]["status"] != Rejected || results[1]["reason"].(error).Error() != "UnexpectedError: error" {
			t.Errorf("Expected second result to be rejected with reason 'error', got %v", results[1])
		}
		if results[2]["status"] != Fulfilled || results[2]["value"] != 3 {
			t.Errorf("Expected third result to be fulfilled with value 3, got %v", results[2])
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise.AllSettled should be fulfilled, but was rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试AllSettled方法 - 空数组
func TestPromiseAllSettledEmptyArray(t *testing.T) {
	t.Parallel()
	el.AllSettled(make([]any, 0)...).Then(func(v any) (any, error) {
		results, ok := v.([]map[string]any)
		if !ok {
			t.Errorf("Expected []map[string]any type, got %T", v)
		}
		if len(results) != 0 {
			t.Errorf("Expected empty array, got %d elements", len(results))
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise.AllSettled should be fulfilled, but was rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试AllSettled方法 - nil数组
func TestPromiseAllSettledNilArray(t *testing.T) {
	t.Parallel()
	el.AllSettled().Then(func(v any) (any, error) {
		t.Errorf("Promise.AllSettled with nil array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "TypeError: nil is not iterable" {
			t.Errorf("Expected error value 'TypeError: nil is not iterable', got %v", v.Error())
		}
		return nil, nil
	})
}

// 测试Any方法 - 有一个Promise成功
func TestPromiseAnyFulfilled(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error1")
		return nil
	})
	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("success")
		return nil
	})
	p3 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error2")
		return nil
	})

	el.Any(p1, p2, p3).Then(func(v any) (any, error) {
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise.Any should be fulfilled, but was rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试Any方法 - 所有Promise都失败
func TestPromiseAnyAllRejected(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error1")
		return nil
	})
	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error2")
		return nil
	})

	el.Any(p1, p2).Then(func(v any) (any, error) {
		t.Errorf("Promise.Any should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		var agg *AggregateError
		ok := errors.As(v, &agg)
		if !ok {
			t.Errorf("Expected *AggregateError, got %T", v)
			return nil, nil
		}
		errs := agg.Unwrap()
		if len(errs) != 2 {
			t.Errorf("Expected 2 errors, got %d", len(errs))
			return nil, nil
		}
		if errs[0].Error() != "UnexpectedError: error1" || errs[1].Error() != "UnexpectedError: error2" {
			t.Errorf("Expected errors ['UnexpectedError: error1', 'UnexpectedError: error2'], got %v", errs)
		}
		return nil, nil
	})
}

// 测试Any方法 - 空数组
func TestPromiseAnyEmptyArray(t *testing.T) {
	t.Parallel()
	el.Any(make([]any, 0)...).Then(func(v any) (any, error) {
		t.Errorf("Promise.Any with empty array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		var agg *AggregateError
		ok := errors.As(v, &agg)
		if !ok {
			t.Errorf("Expected *AggregateError, got %T", v)
			return nil, nil
		}
		errs := agg.Unwrap()
		if len(errs) != 0 {
			t.Errorf("Expected 0 errors, got %d", len(errs))
			return nil, nil
		}
		return nil, nil
	})
}

// 测试Any方法 - nil数组
func TestPromiseAnyNilArray(t *testing.T) {
	t.Parallel()
	el.Any().Then(func(v any) (any, error) {
		t.Errorf("Promise.Any with nil array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "TypeError: nil is not iterable" {
			t.Errorf("Expected error value 'TypeError: nil is not iterable', got %v", v.Error())
		}
		return nil, nil
	})
}

// 测试Some方法 - num<=0
func TestPromiseSomeNumLE0(t *testing.T) {
	t.Parallel()
	el.Some(0, el.Resolve("success")).Then(func(v any) (any, error) {
		t.Errorf("Promise.Some with num<=0 should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "RangeError: num must be greater than 0" {
			t.Errorf("Expected error value 'RangeError: num must be greater than 0', got %v", v.Error())
		}
		return nil, nil
	})
}

// 测试Some方法 - num>proms长度
func TestPromiseSomeNumGTPromsLen(t *testing.T) {
	t.Parallel()
	el.Some(2, el.Resolve("success")).Then(func(v any) (any, error) {
		t.Errorf("Promise.Some with num>proms length should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "RangeError: no enough promises to resolve" {
			t.Errorf("Expected error value 'RangeError: no enough promises to resolve', got %v", v.Error())
		}
		return nil, nil
	})
}

// 测试Some方法 - 3个满足2个
func TestPromiseSome2in3(t *testing.T) {
	t.Parallel()
	p1 := el.Resolve("success1")
	p2 := el.Resolve("success2")
	p3 := el.Reject("failure")

	el.Some(2, p1, p2, p3).Then(func(v any) (any, error) {
		if len(v.([]any)) != 2 {
			t.Errorf("Expected 2 values, got %d", len(v.([]any)))
			return nil, nil
		}
		if v.([]any)[0] != "success1" || v.([]any)[1] != "success2" {
			t.Errorf("Expected values ['success1','success2'], got %v", v.([]any))
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise.Some should be fulfilled, but was rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试Some方法 - 3个拒绝2个
func TestPromiseSome2out3(t *testing.T) {
	t.Parallel()
	p1 := el.Reject("failure1")
	p2 := el.Reject("failure2")
	p3 := el.Resolve("success")

	el.Some(2, p1, p2, p3).Then(func(v any) (any, error) {
		t.Errorf("Promise.Some should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		agg, ok := v.(*AggregateError)
		if !ok {
			t.Errorf("Expected *AggregateError, got %T", v)
			return nil, nil
		}
		errs := agg.Unwrap()
		if len(errs) != 2 {
			t.Errorf("Expected 2 errors, got %d", len(errs))
			return nil, nil
		}
		if errs[0].Error() != "UnexpectedError: failure1" || errs[1].Error() != "UnexpectedError: failure2" {
			t.Errorf("Expected errors ['UnexpectedError: failure1','UnexpectedError: failure2'], got %v", errs)
		}
		return nil, nil
	})
}

// 测试Race方法 - 第一个完成的是成功的Promise
func TestPromiseRaceFulfilled(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		el.SetTimeout(func() {
			reject("error")
		}, 100)
		return nil
	})
	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		el.SetTimeout(func() {
			resolve("success")
		}, 50)
		return nil
	})

	el.Race(p1, p2).Then(func(v any) (any, error) {
		if v != "success" {
			t.Errorf("Expected value 'success', got %v", v)
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise.Race should be fulfilled, but was rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试Race方法 - 第一个完成的是失败的Promise
func TestPromiseRaceRejected(t *testing.T) {
	t.Parallel()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		el.SetTimeout(func() {
			reject("error")
		}, 50)
		return nil
	})
	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		el.SetTimeout(func() {
			resolve("success")
		}, 100)
		return nil
	})

	el.Race(p1, p2).Then(func(v any) (any, error) {
		t.Errorf("Promise.Race should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "UnexpectedError: error" {
			t.Errorf("Expected value 'UnexpectedError: error', got %v", v.Error())
		}
		return nil, nil
	})
}

// 测试Race方法 - 空数组
func TestPromiseRaceEmptyArray(t *testing.T) {
	t.Parallel()
	racePromise := el.Race(make([]any, 0)...)
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
}

// 测试Race方法 - nil数组
func TestPromiseRaceNilArray(t *testing.T) {
	t.Parallel()
	el.Race().Then(func(v any) (any, error) {
		t.Errorf("Promise.Race with nil array should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "TypeError: nil is not iterable" {
			t.Errorf("Expected error value 'TypeError: nil is not iterable', got %v", v.Error())
		}
		return nil, nil
	})
}

// 测试Resolve方法 - 普通值
func TestPromiseResolve(t *testing.T) {
	t.Parallel()
	el.Resolve("value").Then(func(v any) (any, error) {
		if v != "value" {
			t.Errorf("Expected value 'value', got %v", v)
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise.Resolve should be fulfilled, but was rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试Resolve方法 - Promise对象 - fulfilled状态
func TestPromiseResolvePromise(t *testing.T) {
	t.Parallel()
	original := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("original")
		return nil
	})

	p := el.Resolve(original)

	if p != original {
		t.Errorf("Expected Resolve to return the same Promise instance")
	}

	p.Then(func(v any) (any, error) {
		if v != "original" {
			t.Errorf("Expected value 'original', got %v", v)
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Expected state Fulfilled, got Rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试Resolve方法 - Promise对象 - rejected状态
func TestPromiseResolvePromiseRejected(t *testing.T) {
	t.Parallel()
	original := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})

	p := el.Resolve(original)

	if p != original {
		t.Errorf("Expected Resolve to return the same Promise instance")
	}

	p.Then(func(v any) (any, error) {
		t.Errorf("Expected state Rejected, got Fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "UnexpectedError: error" {
			t.Errorf("Expected value 'UnexpectedError: error', got '%s'", v.Error())
		}
		return nil, nil
	})
}

// 测试Reject方法
func TestPromiseReject(t *testing.T) {
	t.Parallel()
	el.Reject("reason").Then(func(v any) (any, error) {
		t.Errorf("Promise.Reject should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "UnexpectedError: reason" {
			t.Errorf("Expected value 'UnexpectedError: reason', got '%s'", v.Error())
		}
		return nil, nil
	})
}

// 测试Try方法 - 成功
func TestPromiseTrySuccess(t *testing.T) {
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

// 测试Try方法 - 失败
func TestPromiseTryError(t *testing.T) {
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
func TestPromiseTryNilFunc(t *testing.T) {
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

// 测试循环引用检测
func TestPromiseCycleDetection(t *testing.T) {
	t.Parallel()
	initial := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("initial")
		return nil
	})

	var p Promise
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

// 测试Thenable对象处理
func TestPromiseThenable(t *testing.T) {
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
}

// 测试多个Then调用的顺序
func TestPromiseMultipleThenOrder(t *testing.T) {
	t.Parallel()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
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

// 测试SetTimeout函数
func TestSetTimeout(t *testing.T) {
	t.Parallel()
	var str string
	el.SetTimeout(func() {
		str = "timeout value"
	}, 100)

	el.SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 80)

	el.SetTimeout(func() {
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	}, 120)
}

// 测试SetTimeout函数 - 取消
func TestSetTimeoutCancel(t *testing.T) {
	t.Parallel()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		id := el.SetTimeout(func() {
			resolve("timeout value")
		}, 100)
		el.ClearTimeout(id)
		return nil
	})

	el.SetTimeout(func() {
		if p.State() != Pending {
			t.Errorf("Expected state Pending, got %v", p.State())
		}
	}, 120)
}

// 测试SetTimeout函数 - 毫秒数为负数
func TestSetTimeoutNegativeMillis(t *testing.T) {
	t.Parallel()
	var str string
	el.SetTimeout(func() {
		str = "timeout value"
	}, -100)

	el.SetTimeout(func() {
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	}, 20)
}

// 测试SetTimeout函数 - 毫秒数为 0
func TestSetTimeoutZeroMillis(t *testing.T) {
	t.Parallel()
	var str string
	el.SetTimeout(func() {
		str = "timeout value"
	}, 0)

	el.SetTimeout(func() {
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	}, 0)
}

// 测试SetTimeout函数 - 长延迟
func TestSetTimeoutLongDelay(t *testing.T) {
	t.Parallel()
	var str string
	el.SetTimeout(func() {
		str = "timeout value"
	}, 1000)

	el.SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 980)

	el.SetTimeout(func() {
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	}, 1020)
}

// 测试SetInterval函数
func TestSetInterval(t *testing.T) {
	t.Parallel()
	var str string
	var count int
	var id int

	id = el.SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			el.ClearInterval(id)
		}
	}, 200)

	el.SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 180)

	el.SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 220)

	el.SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 380)

	el.SetTimeout(func() {
		if str != "interval interval " {
			t.Errorf("Expected str 'interval interval ', got %s", str)
		}
	}, 420)

	el.SetTimeout(func() {
		if str != "interval interval " {
			t.Errorf("Expected str 'interval interval ', got %s", str)
		}
	}, 580)

	el.SetTimeout(func() {
		if str != "interval interval interval " {
			t.Errorf("Expected str 'interval interval interval ', got %s", str)
		}
	}, 620)
}

// 测试SetInterval函数 - 取消 - 第1次执行
func TestSetIntervalCancelFirst(t *testing.T) {
	t.Parallel()
	var str string
	var count int
	var id int

	id = el.SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			el.ClearInterval(id)
		}
	}, 200)

	el.SetTimeout(func() {
		el.ClearInterval(id)
	}, 20)

	el.SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 240)
}

// 测试SetInterval函数 - 取消 - 非首次执行
func TestSetIntervalCancelNonFirst(t *testing.T) {
	t.Parallel()
	var str string
	var count int
	var id int

	id = el.SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			el.ClearInterval(id)
		}
	}, 200)

	el.SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
		el.ClearInterval(id)
	}, 220)

	el.SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 620)
}

// 测试SetInterval函数 - 长延迟
func TestSetIntervalLongDelay(t *testing.T) {
	t.Parallel()
	var str string
	var id int

	count := 0
	id = el.SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			el.ClearInterval(id)
		}
	}, 1000)

	el.SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 980)
	el.SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 1020)

	el.SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 1980)
	el.SetTimeout(func() {
		if str != "interval interval " {
			t.Errorf("Expected str 'interval interval ', got %s", str)
		}
	}, 2020)

	el.SetTimeout(func() {
		if str != "interval interval " {
			t.Errorf("Expected str 'interval interval ', got %s", str)
		}
	}, 2980)
	el.SetTimeout(func() {
		if str != "interval interval interval " {
			t.Errorf("Expected str 'interval interval interval ', got %s", str)
		}
	}, 3020)
}

// 测试Await - 成功
func TestAwaitSuccess(t *testing.T) {
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

// 测试Await - timeout不是正数
func TestAwaitTimeoutNotPositive(t *testing.T) {
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("success")
		return nil
	})

	expected := "RangeError: await timeout must be greater than 0"
	_, err := el.Await(p, -100)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got %s", expected, err)
	}
}

// 测试Await - 超时
func TestAwaitTimeout(t *testing.T) {
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		el.SetTimeout(func() {
			resolve("success")
		}, 100)
		return nil
	})

	_, err := el.Await(p, 50)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if err.Error() != "TimeoutError: await timeout" {
		t.Errorf("Expected error 'TimeoutError: await timeout', got %s", err)
	}
}

// 测试Await - 拒绝的Promise
func TestAwaitRejectedPromise(t *testing.T) {
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})

	_, err := el.Await(p, 50)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if err.Error() != "UnexpectedError: error" {
		t.Errorf("Expected error 'UnexpectedError: error', got %s", err)
	}
}

// 测试Delay函数
func TestDelay(t *testing.T) {
	el2 := StartClassicEventLoop()
	val := "value"

	if _, err := el2.Await(el2.Delay(val, 50), 40); err == nil {
		// timeout
		t.Errorf("Expected 'TimeoutError: await timeout', got nil")
	}

	res, err := el2.Await(el2.Delay(val, 50), 100)
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if res != val {
		t.Errorf("Expected '%s', got %s", val, res)
	}

	p := el2.NewPromise(func(resolve, reject func(v any)) error {
		el2.SetTimeout(func() {
			resolve("success")
		}, 50)
		return nil
	})
	if _, err = el2.Await(el2.Delay(p, 50), 90); err == nil {
		// timeout
		t.Errorf("Expected 'TimeoutError: await timeout', got nil")
	}

	p2 := el2.NewPromise(func(resolve, reject func(v any)) error {
		el2.SetTimeout(func() {
			resolve("success")
		}, 50)
		return nil
	})
	if res, _ := el2.Await(el2.Delay(p2, 50), 150); res != "success" {
		t.Errorf("Expected 'success', got %s", res)
	}

	_ = el2.Close()
}

// 测试异步调用顺序 - 微任务
func TestAsyncCallOrderMicro(t *testing.T) {
	t.Parallel()
	result := ""
	el.SetTimeout(func() {
		var res func(any)
		p := el.Resolve("success")
		p.Then(func(v any) (any, error) {
			result += "[B]"
			return nil, nil
		}, nil).Then(func(v any) (any, error) {
			result += "[D]"
			return nil, nil
		}, nil)
		el.NewPromise(func(resolve, reject func(v any)) error {
			result += "[A]"
			res = resolve
			return nil
		}).Then(func(v any) (any, error) {
			result += "[E]"
			return nil, nil
		}, nil)
		res(p)
		el.QueueMicrotask(func() {
			result += "[C]"
		})
	}, 0)

	el.SetTimeout(func() {
		if result != "[A][B][C][D][E]" {
			t.Errorf("Expected result1 '[A][B][C][D][E]', got %s", result)
		}
	}, 0)
}

// 测试异步调用顺序 - 宏任务
func TestAsyncCallOrderMacro(t *testing.T) {
	t.Parallel()
	result := ""
	el.SetTimeout(func() {
		result += "[A]"
		el.SetTimeout(func() {
			result += "[C]"
		}, 20)
	}, 30)
	// 无延迟
	el.SetTimeout(func() {
		result += "[B]"
	}, 50)

	el.SetTimeout(func() {
		if result != "[A][B][C]" {
			t.Errorf("Expected result1 '[A][B][C]', got %s", result)
		}
	}, 60)
}

// 测试异步调用顺序 - 宏任务 - 有延迟
func TestAsyncCallOrderMacroDelay(t *testing.T) {
	t.Parallel()
	result := ""

	el.SetTimeout(func() {
		result += "[A]"
		el.SetTimeout(func() {
			result += "[C]"
		}, 20)
	}, 30)
	// 有延迟 - 看情况调大数字，go实在太快了，区区循环嗖一下就完成了
	for i := range 12345678 {
		_ = int64(i) * 1234 / 2234 % 3234
	}
	el.SetTimeout(func() {
		result += "[B]"
	}, 50)

	el.SetTimeout(func() {
		if result != "[A][C][B]" {
			t.Errorf("Expected result2 '[A][C][B]', got %s", result)
		}
	}, 60)
}

// 测试异步调用顺序 - 混合模式
func TestAsyncCallOrderMixed(t *testing.T) {
	t.Parallel()
	var result string

	el.SetTimeout(func() {
		result += "[01]"

		el.SetTimeout(func() {
			result += "-[28]"

			el.Reject("error").Catch(func(v error) (any, error) {
				result += "-[29]"
				return nil, nil
			})

			el.QueueMicrotask(func() {
				result += "-[30]"
				/*
				   根据ES规范，37应该出现在40后面，但在浏览器中，由于0ms小于最低延迟时间（通常是4ms），
				   导致37出现在36和38之间，即浏览器执行是01-42完全连续（浏览器对0ms延迟做了特别处理）。
				   注：37和31、35、36、38、39的理论触发时间相同，根据ES规范，先注册先执行，37是最后注册的，
				   但由于执行延迟（见 [TestAsyncCallOrderMacroDelay]），37可能出现在上述5个任务的任意一个之前，是正常现象。
				*/
				el.SetTimeout(func() {
					result += "-[37]"
				}, 0)
			})

			el.SetTimeout(func() {
				result += "-[41]"
			}, 30)
		}, 50)

		p1 := el.Resolve(nil)
		p1.Then(func(v any) (any, error) {
			result += "-[04]"

			el.Resolve(nil).Then(func(v any) (any, error) {
				result += "-[10]"
				return nil, nil
			}, nil)

			el.QueueMicrotask(func() {
				result += "-[11]"
				el.SetTimeout(func() {
					result += "-[38]"
				}, 50)
			})

			el.SetTimeout(func() {
				result += "-[26]"
			}, 0)

			return nil, nil
		}, nil)

		p2 := el.Resolve(nil)
		p2.Then(func(v any) (any, error) {
			result += "-[05]"
			return nil, nil
		}, nil).Then(func(v any) (any, error) {
			result += "-[12]"

			el.Resolve(nil).Then(func(v any) (any, error) {
				result += "-[19]"
				return nil, nil
			}, nil).Then(func(v any) (any, error) {
				result += "-[24]"
				return nil, nil
			}, nil)

			el.QueueMicrotask(func() {
				result += "-[20]"
			})

			el.SetTimeout(func() {
				result += "-[39]"

				el.QueueMicrotask(func() {
					result += "-[40]"
				})
			}, 50)

			return nil, nil
		}, nil).Then(func(v any) (any, error) {
			result += "-[21]"
			return nil, nil
		}, nil)

		p3 := el.NewPromise(func(resolve, reject func(v any)) error {
			result += "-[02]"
			resolve(4)
			return nil
		})

		p3.Then(func(v any) (any, error) {
			result += "-[06]"

			el.Reject("error").Catch(func(v error) (any, error) {
				result += "-[13]"

				el.Resolve(nil).Then(func(v any) (any, error) {
					result += "-[22]"
					return nil, nil
				}, nil)

				return nil, nil
			})

			el.QueueMicrotask(func() {
				result += "-[14]"
			})

			el.SetTimeout(func() {
				result += "-[35]"
			}, 50)

			return nil, nil
		}, nil)

		p3.Then(func(v any) (any, error) {
			result += "-[07]"

			el.Resolve(nil).Finally(func() (any, error) {
				result += "-[15]"
				return nil, nil
			})

			el.QueueMicrotask(func() {
				result += "-[16]"
			})

			el.SetTimeout(func() {
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
		id = el.SetInterval(func() {
			result += "-[31]"

			if count >= 2 {
				el.ClearInterval(id)
			}
			count++

			el.Resolve(nil).Finally(func() (any, error) {
				result += "-[32]"

				el.Resolve(nil).Finally(func() (any, error) {
					result += "-[34]"
					return nil, nil
				})

				return nil, nil
			})

			el.QueueMicrotask(func() {
				result += "-[33]"
			})

			el.SetTimeout(func() {
				result += "-[42]"
			}, 30)
		}, 50)

		el.QueueMicrotask(func() {
			result += "-[09]"

			p := el.Resolve(nil)
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

			el.QueueMicrotask(func() {
				result += "-[18]"
			})

			el.SetTimeout(func() {
				result += "-[27]"
			}, 0)
		})

		result += "-[03]"
	}, 0)

	el.SetTimeout(func() {

		expected := "[01]-[02]-[03]-[04]-[05]-[06]-[07]-[08]-[09]-[10]-[11]-[12]-[13]-[14]-[15]-[16]-[17]-[18]-[19]-[20]-[21]-[22]-[23]-[24]-[25]-[26]-[27]-[28]-[29]-[30]-[31]-[32]-[33]-[34]-[35]-[36]-[38]-[39]-[40]-[37]-[41]-[42]-[31]-[32]-[33]-[34]-[42]"
		if result != expected {
			t.Errorf("Expected result:\n%s\n\nGot:\n%s", expected, result)
		}
	}, 360)
}
