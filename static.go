package promise

import (
	"sync/atomic"
	"time"
)

/*
All 等待所有输入解决。
  - 如果 inputs 的所有元素都成功解决，新 Promise 也会成功解决，且解决值为一个包含所有元素解决值的数组；
  - 如果任何一个元素被拒绝，新 Promise 也会被拒绝，且拒绝理由为第一个被拒绝的元素的拒绝理由。
*/
func All(inputs ...any) Promise {
	if inputs == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(inputs) == 0 {
		return Resolve(make([]any, 0))
	}

	return New(func(resolve, reject func(v any)) any {
		results := make([]any, len(inputs))
		var count int32 = 0

		for index, item := range inputs {
			prom := Resolve(item)
			prom.Then(func(v any) (any, any) {
				results[index] = v
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(inputs) {
					resolve(results)
				}
				return nil, nil
			}, func(reason any) (any, any) {
				reject(reason)
				return nil, nil
			})
		}

		return nil
	})
}

/*
AllSettled 等待所有 Promise 完成（无论成功失败）。
  - 新 Promise 会在所有 Promise 完成后解决，解决值为一个包含所有 Promise 完成状态和结果的数组。
*/
func AllSettled(inputs ...any) Promise {
	if inputs == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(inputs) == 0 {
		return Resolve(make([]map[string]any, 0))
	}

	return New(func(resolve, reject func(v any)) any {
		type result struct {
			Status string
			Value  any
			Reason any
		}

		length := len(inputs)
		results := make([]result, length)
		var count int32 = 0
		for index, item := range inputs {
			prom := Resolve(item)
			settleData := func() {
				finalResults := make([]map[string]any, length)
				for i, r := range results {
					finalResults[i] = make(map[string]any)
					finalResults[i]["status"] = r.Status
					if r.Status == Fulfilled {
						finalResults[i]["value"] = r.Value
					} else {
						finalResults[i]["reason"] = r.Reason
					}
				}
				resolve(finalResults)
			}
			prom.Then(func(v any) (any, any) {
				results[index] = result{Status: Fulfilled, Value: v}
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == length {
					settleData()
				}
				return nil, nil
			}, func(reason any) (any, any) {
				results[index] = result{Status: Rejected, Reason: reason}
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == length {
					settleData()
				}
				return nil, nil
			})
		}

		return nil
	})
}

/*
Any 等待 inputs 中第一个成功解决的元素。
  - 如果任何一个 Promise 解决，新 Promise 也会被解决，且解决值为第一个被解决的解决值。
  - 如果所有 Promise 都被拒绝，新 Promise 也会被拒绝，且拒绝理由为一个包含所有 Promise 拒绝理由的 map，
    其顺序为 Promise 数组中的顺序。
*/
func Any(inputs ...any) Promise {
	if inputs == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(inputs) == 0 {
		result := make(map[string]any)
		result["errors"] = make([]any, 0)
		result["stack"] = "AggregateError: All promises were rejected"
		result["message"] = "All promises were rejected"
		return Reject(result)
	}

	return New(func(resolve, reject func(v any)) any {
		length := len(inputs)
		reasons := make([]any, length)

		var count int32 = 0

		for index, item := range inputs {
			prom := Resolve(item)
			prom.Then(func(v any) (any, any) {
				resolve(v)
				return nil, nil
			}, func(reason any) (any, any) {
				reasons[index] = reason
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == length {
					result := make(map[string]any)
					result["errors"] = reasons
					result["stack"] = "AggregateError: All promises were rejected"
					result["message"] = "All promises were rejected"
					reject(result)
				}
				return nil, nil
			})
		}

		return nil
	})
}

/*
Async 将代码作为一个异步任务执行。
将会开启新的go程，执行 fn 函数，可用于执行耗时任务，避免阻塞事件循环。

返回一个 Promise 实例，并在 fn 函数执行完成后变为已决状态，
若 fn 函数抛出异常，则 Promise 实例会被拒绝，且拒绝理由为异常信息。
*/
func Async(fn func()) Promise {
	if fn == nil {
		return Reject("TypeError: fn must be a function")
	}

	return New(func(resolve, reject func(v any)) any {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					reject(err)
				}
			}()
			fn()
			resolve(nil)
		}()
		return nil
	})
}

/*
Await 等待 Promise 完成，并设定超时时间，以免无限等待。

  - prom 需要等待的 Promise 实例，如果不是 Promise 实例，则会被包装成 Promise。
  - timeout 超时时间，单位为毫秒。

返回值：v 是已决值，err 是拒绝理由，当 err 存在时，代表 Promise 被拒绝。
*/
func Await(prom any, timeout int64) (v any, err any) {
	if timeout <= 0 {
		return nil, "await timeout must be greater than 0"
	}

	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	defer timer.Stop()

	prom2, ok := prom.(Promise)
	if !ok {
		return prom, nil
	}
	select {
	case <-prom2.Done():
		res := prom2.Result()
		if prom2.State() == Rejected {
			err = res
		} else {
			v = res
		}
	case <-timer.C:
		err = "TimeoutError: await timeout"
	}

	return
}

/*
Each 按顺序等待数组中的每个元素完成，每个元素的完成结果会被传递给回调函数。
如果迭代器返回一个 Promise，则会等待该 Promise 完成后再继续迭代；
如果当前迭代对象是 Promise，则会等待其完成后再继续迭代；
迭代过程中遇到任何一个 Promise 被拒绝，新 Promise 也会以同样的理由被拒绝。

由于迭代器的输出会被丢弃，因此适合副作用操作，如打印日志等。

  - it 对每个元素进行操作的函数，接受三个参数：item（当前元素）、index（当前元素的索引）、arrLen（数组长度）。
  - inputs 需要迭代的输入。

返回一个 Promise，其状态可以是：
  - 已解决（Fulfilled）：如果所有迭代都成功解决，已决值是包含原始输入已决值的数组。
  - 已拒绝（Rejected）：如果迭代过程中任何一个 Promise 被拒绝。
*/
func Each(it func(item any, index int, arrLen int) any, inputs ...any) Promise {
	if inputs == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(inputs) == 0 {
		return Resolve(make([]any, 0))
	}
	if it == nil {
		return Reject("TypeError: nil is not a function")
	}

	prom := Resolve("start")
	arrLen := len(inputs)
	result := make([]any, arrLen)
	for index, item := range inputs {
		prom = prom.
			Then(func(any) (any, any) {
				return item, nil
			}, nil).
			Then(func(v any) (any, any) {
				result[index] = v
				return it(v, index, arrLen), nil
			}, nil)
	}

	return prom.
		Then(func(any) (any, any) {
			return result, nil
		}, nil)
}

/*
Delay 返回一个新的 Promise，其状态会在延迟时间后被解决。

  - prom 将会使用的已决值，如果 prom 是 Promise 实例，则会等待其完成后才开始计时；
    如果是一个已拒绝的 Promise，则会立即拒绝新 Promise。
  - timeout 延迟时间，单位为毫秒。
*/
func Delay(prom any, millis int64) Promise {
	return New(func(resolve, reject func(v any)) any {
		Resolve(prom).Then(func(v2 any) (any, any) {
			SetTimeout(func() {
				resolve(v2)
			}, millis)
			return nil, nil
		}, func(r any) (any, any) {
			reject(r)
			return nil, nil
		})
		return nil
	})
}

/*
Filter 过滤数组中的元素，返回一个新的 Promise，其状态可以是：
  - 已解决（Fulfilled）：如果所有 Promise 都成功解决，已决值是过滤后的数组。
  - 已拒绝（Rejected）：如果任何一个 Promise 被拒绝。

本质上是 Map + Array.Filter 的快捷方式。
*/
func Filter(filter func(item any) bool, inputs ...any) Promise {
	return Map(func(item any) any {
		return All(item, filter(item))
	}, inputs...).
		Then(func(v any) (any, any) {
			values := v.([]any)
			result := make([]any, 0)
			for _, item := range values {
				tuple := item.([]any)
				if tuple[1].(bool) {
					result = append(result, tuple[0])
				}
			}
			return result, nil
		}, nil)
}

/*
Map 对输入数组中的每个元素应用一个函数，返回一个新的 Promise 数组，新数组的每个元素都是原数组对应元素应用函数后的结果。
  - mapper 对每个元素进行操作的函数，接受一个参数 item 并返回一个新值。
  - inputs 被映射的输入。

返回一个 Promise，其状态可以是：
  - 已解决（Fulfilled）：如果所有 Promise 都成功解决，且每个 Promise 的解决值都被 mapper 处理后得到新值。
  - 已拒绝（Rejected）：如果任何一个 Promise 被拒绝。
*/
func Map(mapper func(item any) any, inputs ...any) Promise {
	if inputs == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(inputs) == 0 {
		return Resolve(make([]any, 0))
	}
	if mapper == nil {
		return Reject("TypeError: nil is not a function")
	}

	result := make([]any, len(inputs))
	for index, item := range inputs {
		result[index] = Resolve(item).Then(func(v any) (any, any) {
			return mapper(v), nil
		}, nil)
	}

	return All(result...)
}

/*
PromiseWithResolvers 创建一个新的 Promise 实例，同时返回 resolve 和 reject 函数，
对应于传入给 Promise() 构造函数执行器的两个参数。

这使得可以在 Promise 外部手动解决或拒绝 Promise，详见 [MDN]。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/withResolvers#%E6%8F%8F%E8%BF%B0
*/
func PromiseWithResolvers() (Promise, func(any), func(any)) {
	var resolve, reject func(any)
	p := New(func(res func(any), rej func(any)) any {
		resolve = res
		reject = rej
		return nil
	})
	return p, resolve, reject
}

/*
Promisify 将一个返回值格式为 (result, error) 的函数转换为返回 Promise 的函数，
原函数的第二个返回值将被视为 Promise 的拒绝理由。

如果原函数 panic，新 Promise 会被拒绝，拒绝理由为 panic 值。
*/
func Promisify[A any, V any](fn func(args ...A) (V, error)) func(args ...A) Promise {
	return func(args ...A) Promise {
		return New(func(resolve, reject func(v any)) any {
			defer func() {
				if r := recover(); r != nil {
					reject(r)
				}
			}()
			res, err := fn(args...)
			if err != nil {
				reject(err)
			} else {
				resolve(res)
			}
			return nil
		})
	}
}

/*
Race 等待第一个 Promise 完成。
  - 新 Promise 会在第一个 Promise 完成后解决或拒绝，解决值或拒绝理由跟随第一个完成的 Promise。
*/
func Race(inputs ...any) Promise {
	if inputs == nil {
		return Reject("TypeError: nil is not iterable")
	}

	return New(func(resolve, reject func(v any)) any {
		if len(inputs) == 0 {
			return nil
		}

		for _, item := range inputs {
			prom := Resolve(item)
			prom.Then(func(v any) (any, any) {
				resolve(v)
				return nil, nil
			}, func(reason any) (any, any) {
				reject(reason)
				return nil, nil
			})
		}

		return nil
	})
}

/*
Reduce 对数组中的每个元素应用一个函数，将其结果累积到一个累加器中，最后返回累加器的值。

  - reducer 对每个元素进行操作的函数，接受两个参数 acc（累加器）和 cur（当前元素），
    返回新的累加器值。
  - initial 初始累加器值。
  - inputs 被操作的数组。

返回一个 Promise，其状态可以是：
  - 已解决（Fulfilled）：如果所有 Promise 都成功解决，且每个 Promise 的解决值都被 reducer 处理后得到新值。
  - 已拒绝（Rejected）：如果任何一个 Promise 被拒绝。

特殊情况：
  - 如果 inputs 为空数组，直接返回初始 initial；
  - 如果 initial 为 nil，且 inputs 只有一个元素，直接返回该元素。
*/
func Reduce(reducer func(acc any, cur any) any, initial any, inputs ...any) Promise {
	init := Resolve(initial)

	if len(inputs) == 0 {
		return init
	}

	return init.
		Then(func(v any) (any, any) {
			if v == nil && len(inputs) == 1 {
				return inputs[0], nil
			}

			acc := v
			result := Each(func(item any, index int, arrLen int) any {
				acc = reducer(acc, item)
				return nil
			}, inputs...).
				Then(func(v any) (any, any) {
					return acc, nil
				}, nil)
			return result, nil
		}, nil)

}

/*
Reject 返回一个已拒绝的 Promise，拒绝理由为指定值，详见 [MDN]。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/reject
*/
func Reject(reason any) Promise {
	return New(func(resolve, reject func(v any)) any {
		reject(reason)
		return nil
	})
}

/*
Resolve 返回一个已解决的 Promise，解决值为指定值 value。

如果 value 已经是 Promise，则直接返回该 Promise，详见 [MDN]。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/resolve
*/
func Resolve(value any) Promise {
	if prom, ok := value.(Promise); ok {
		return prom
	}

	return New(func(resolve, reject func(v any)) any {
		resolve(value)
		return nil
	})
}

/*
Some 等待 inputs 中前 num 个元素解决。
  - 如果 num 个元素解决，新 Promise 也会被解决，且解决值为一个包含所有元素解决值的数组，
    其顺序为被解决的顺序。
  - 如果太多元素被拒绝，以至于新 Promise 永远无法满足，那么新 Promise 会立即被拒绝，
    且拒绝理由为一个包含所有元素拒绝理由的 map，其顺序为被拒绝的顺序。

注意与 [Any] 的不同，不仅是解决值的格式不同，拒绝理由的顺序也不同。
*/
func Some(num int, inputs ...any) Promise {
	if inputs == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if num <= 0 {
		return Reject("RangeError: num must be greater than 0")
	}
	if len(inputs) == 0 {
		result := make(map[string]any)
		result["errors"] = make([]any, 0)
		result["stack"] = "AggregateError: All promises were rejected"
		result["message"] = "All promises were rejected"
		return Reject(result)
	}
	if num > len(inputs) {
		return Reject("RangeError: not enough promises to resolve")
	}

	return New(func(resolve, reject func(v any)) any {
		threshold := len(inputs) - num + 1
		values := make([]any, 0, num)
		reasons := make([]any, 0, threshold)
		var resCount int32 = 0
		var rejCount int32 = 0

		for _, item := range inputs {
			prom := Resolve(item)
			prom.Then(func(v any) (any, any) {
				values = append(values, v)
				if newCount := atomic.AddInt32(&resCount, 1); int(newCount) == num {
					resolve(values)
				}
				return nil, nil
			}, func(reason any) (any, any) {
				reasons = append(reasons, reason)
				if newCount := atomic.AddInt32(&rejCount, 1); int(newCount) == threshold {
					result := make(map[string]any)
					result["errors"] = reasons
					result["stack"] = "AggregateError: Too many promises were rejected"
					result["message"] = "Too many promises were rejected"
					reject(result)
				}
				return nil, nil
			})
		}
		return nil
	})
}

/*
Try 接受一个任意类型的回调函数（无论其是同步或异步，返回结果或抛出异常），并将其结果封装成一个 Promise，详见 [MDN]。
  - fn 任意类型的回调函数，接受任意数量的参数，函数返回值格式与 [ThenCallback] 相同。
  - args 将要传递给 fn 函数的参数列表。

返回一个 Promise，其状态可以是：
  - 已解决（Fulfilled）：如果 fn 函数返回一个非错误值。
  - 已拒绝（Rejected）：如果 fn 函数返回一个错误值。
  - 异步解决或拒绝：如果 fn 函数返回一个 Promise，新 Promise 会吸收该 Promise 的状态。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/try
*/
func Try(fn func(...any) (any, any), args ...any) Promise {
	return New(func(resolve, reject func(v any)) any {
		if fn == nil {
			reject("Promise executor must be a function")
			return nil
		}

		result, err := fn(args...)
		if err != nil {
			reject(result)
			return nil
		}
		resolve(result)
		return nil
	})
}
