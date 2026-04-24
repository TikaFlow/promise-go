package promise_test

import (
	"testing"
	"time"
)

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
	}, 100)

	time.Sleep(time.Second)
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
	for i := range 12345 {
		time.Sleep(time.Nanosecond)
		_ = int64(i) * 2234 / 3234 % 4234
	}
	el.SetTimeout(func() {
		result += "[B]"
	}, 50)

	el.SetTimeout(func() {
		if result != "[A][C][B]" {
			t.Errorf("Expected result2 '[A][C][B]', got %s", result)
		}
	}, 100)

	time.Sleep(time.Second)
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
		}, nil).
			Then(func(v any) (any, error) {
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
			t.Errorf("Expected result '[A][B][C][D][E]', got %s", result)
		}
	}, 0)

	time.Sleep(time.Second)
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
		part1 := "[01]-[02]-[03]-[04]-[05]-[06]-[07]-[08]-[09]-[10]-[11]-[12]-[13]-[14]-[15]-[16]-[17]-[18]-[19]-[20]-[21]-[22]-[23]-[24]-[25]-[26]-[27]-[28]-[29]-[30]-"
		part20 := "[31]-[32]-[33]-[34]-[35]-[36]-[38]-[39]-[40]-[37]-"
		part21 := "[37]-[31]-[32]-[33]-[34]-[35]-[36]-[38]-[39]-[40]-"
		part22 := "[31]-[32]-[33]-[34]-[37]-[35]-[36]-[38]-[39]-[40]-"
		part23 := "[31]-[32]-[33]-[34]-[35]-[37]-[36]-[38]-[39]-[40]-"
		part24 := "[31]-[32]-[33]-[34]-[35]-[36]-[37]-[38]-[39]-[40]-"
		part25 := "[31]-[32]-[33]-[34]-[35]-[36]-[38]-[37]-[39]-[40]-"
		part3 := "[41]-[42]-[31]-[32]-[33]-[34]-[42]"
		expected0 := part1 + part20 + part3
		expected1 := part1 + part21 + part3
		expected2 := part1 + part22 + part3
		expected3 := part1 + part23 + part3
		expected4 := part1 + part24 + part3
		expected5 := part1 + part25 + part3
		if result != expected0 && result != expected1 && result != expected2 && result != expected3 && result != expected4 && result != expected5 {
			t.Errorf("Expected result is ordered but got:\n%s", result)
		}
	}, 360)

	time.Sleep(time.Second)
}
