package promise_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 测试All方法 - 空数组
func TestAllEmptyArray(t *testing.T) {
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

// 测试All方法 - 所有Promise都成功
func TestAllFulfilled(t *testing.T) {
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

// 测试All方法 - nil数组
func TestAllNilArray(t *testing.T) {
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

// 测试All方法 - 含有非Promise对象
func TestAllNotPromise(t *testing.T) {
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
func TestAllRejected(t *testing.T) {
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

// 测试AllSettled方法
func TestAllSettled(t *testing.T) {
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
func TestAllSettledEmptyArray(t *testing.T) {
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
func TestAllSettledNilArray(t *testing.T) {
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

// 测试Any方法 - 所有Promise都失败
func TestAnyAllRejected(t *testing.T) {
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
func TestAnyEmptyArray(t *testing.T) {
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

// 测试Any方法 - 有一个Promise成功
func TestAnyFulfilled(t *testing.T) {
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

// 测试Any方法 - nil数组
func TestAnyNilArray(t *testing.T) {
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

// 测试Each方法 - 迭代器返回promise
func TestEachIteratorReturnsPromise(t *testing.T) {
	t.Parallel()
	el.Each(func(item any, index int, arrLen int) any {
		return el.Resolve(item.(int) * 2)
	}, 1, 2, 3).
		Then(func(v any) (any, error) {
			arr := v.([]any)
			if len(arr) != 3 {
				t.Errorf("Expected array length 3, got %d", len(arr))
			}
			if arr[0] != 1 || arr[1] != 2 || arr[2] != 3 {
				t.Errorf("Expected [1, 2, 3], got %v", arr)
			}
			return nil, nil
		}, nil)

	time.Sleep(2 * time.Second)
}

// 测试Each方法 - 空数组
func TestEachEmptyArray(t *testing.T) {
	t.Parallel()
	el.Each(func(item any, index int, arrLen int) any {
		return item
	}).
		Then(func(v any) (any, error) {
			if len(v.([]any)) != 0 {
				t.Errorf("Expected empty array, got %v", v)
			}
			return nil, nil
		}, nil)

	time.Sleep(2 * time.Second)
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

// 测试Filter方法 - 空数组
func TestFilterEmptyArray(t *testing.T) {
	t.Parallel()
	filter := func(item any) bool {
		return true
	}
	el.Filter(filter).
		Then(func(v any) (any, error) {
			if len(v.([]any)) != 0 {
				t.Errorf("Expected empty array, got %v", v)
			}
			return nil, nil
		}, nil)

	time.Sleep(2 * time.Second)
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

// 测试Race方法 - 空数组
func TestRaceEmptyArray(t *testing.T) {
	t.Parallel()
	racePromise := el.Race(make([]any, 0)...)

	_, err := el.Await(racePromise, 100)
	if err == nil || err.Error() != "TimeoutError: await timeout" || racePromise.State() != Pending {
		t.Errorf("Expected state Pending for empty array Race, got %v", err)
	}
}

// 测试Race方法 - 第一个完成的是成功的Promise
func TestRaceFulfilled(t *testing.T) {
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

	time.Sleep(2 * time.Second)
}

// 测试Race方法 - nil数组
func TestRaceNilArray(t *testing.T) {
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

// 测试Race方法 - 第一个完成的是失败的Promise
func TestRaceRejected(t *testing.T) {
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

	time.Sleep(2 * time.Second)
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

// 测试Some方法 - 3个满足2个
func TestSome2in3(t *testing.T) {
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
func TestSome2out3(t *testing.T) {
	t.Parallel()
	p1 := el.Reject("failure1")
	p2 := el.Reject("failure2")
	p3 := el.Resolve("success")

	el.Some(2, p1, p2, p3).Then(func(v any) (any, error) {
		t.Errorf("Promise.Some should be rejected, but was fulfilled with %v", v)
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
		if errs[0].Error() != "UnexpectedError: failure1" || errs[1].Error() != "UnexpectedError: failure2" {
			t.Errorf("Expected errors ['UnexpectedError: failure1','UnexpectedError: failure2'], got %v", errs)
		}
		return nil, nil
	})
}

// 测试Some方法 - num>proms长度
func TestSomeNumGTPromsLen(t *testing.T) {
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

// 测试Some方法 - num<=0
func TestSomeNumLE0(t *testing.T) {
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
