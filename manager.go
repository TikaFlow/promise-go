package promise

import (
	"sync"
	"sync/atomic"
	"time"
)

var (
	taskQuene      chan func()   = make(chan func(), 1024*10)
	microtaskQuene chan func()   = make(chan func(), 1024*10)
	originTimeout  time.Duration = time.Millisecond * 512
	timeout        time.Duration = originTimeout
	margin         time.Duration = time.Millisecond * 128
	loopTimer      *time.Timer   = time.NewTimer(timeout + margin)
	done           chan struct{} = make(chan struct{})
	magic          time.Duration = 86413 * time.Second
	granted        sync.Once
)

func init() {
	go scheduleTask()
	go startEventLoop()
}

func closeFn() {
	close(done)
}

/*
New 创建一个新的 Promise 实例。
  - exec 执行器函数，用于定义 Promise 的异步操作。
*/
func New(exec Executor) Promise {
	if exec == nil {
		panic("Promise executor must be a function")
	}

	prom := &promiseImpl{
		state:           Pending,
		result:          nil,
		settledHandlers: make(chan *handler, 128),
		settled:         make(chan struct{}),
	}

	res := func(data any) {
		prom.resolved.Do(func() {
			resolve(prom, data)
		})
	}
	rej := func(reason any) {
		prom.resolved.Do(func() {
			reject(prom, reason)
		})
	}

	if err := exec(res, rej); err != nil {
		rej(err)
	}
	return prom
}

/*
All 等待所有 Promise 解决。
  - 如果所有 Promise 都成功解决，新 Promise 也会成功解决，且解决值为一个包含所有 Promise 解决值的数组；
  - 如果任何一个 Promise 被拒绝，新 Promise 也会被拒绝，且拒绝理由为第一个被拒绝的 Promise 的拒绝理由。
*/
func All(proms []Promise) Promise {
	if proms == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(proms) == 0 {
		return Resolve(make([]any, 0))
	}

	return New(func(resolve, reject func(v any)) error {
		results := make([]any, len(proms))
		var count int32 = 0
		rejected := false

		QueueMicrotask(func() {
			for i, prom := range proms {
				go func(index int, p Promise) {
					state, value := Wait(p)

					if rejected {
						return
					}

					if state == Rejected {
						rejected = true
						reject(value)
						return
					}

					results[index] = value

					if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) && !rejected {
						resolve(results)
					}
				}(i, prom)
			}
		})

		return nil
	})
}

/*
AllSettled 等待所有 Promise 完成（无论成功失败）。
  - 新 Promise 会在所有 Promise 完成后解决，解决值为一个包含所有 Promise 完成状态和结果的数组。
*/
func AllSettled(proms []Promise) Promise {
	if proms == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(proms) == 0 {
		return Resolve(make([]map[string]any, 0))
	}

	return New(func(resolve, reject func(v any)) error {
		type result struct {
			Status string
			Value  any
			Reason any
		}

		results := make([]result, len(proms))
		var count int32 = 0
		QueueMicrotask(func() {
			for i, prom := range proms {
				go func(index int, p Promise) {
					state, value := Wait(p)

					if state == Fulfilled {
						results[index] = result{Status: Fulfilled, Value: value}
					} else {
						results[index] = result{Status: Rejected, Reason: value}
					}

					if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) {
						finalResults := make([]map[string]any, len(results))
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
				}(i, prom)
			}
		})

		return nil
	})
}

/*
Any 等待第一个 Promise 解决。
  - 如果任何一个 Promise 解决，新 Promise 也会被解决，且解决值为第一个被解决的 Promise 的解决值。
  - 如果所有 Promise 都被拒绝，新 Promise 也会被拒绝，且拒绝理由为一个包含所有 Promise 拒绝理由的 map。
*/
func Any(proms []Promise) Promise {
	if proms == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(proms) == 0 {
		result := make(map[string]any)
		result["errors"] = make([]any, 0)
		result["stack"] = "AggregateError: All promises were rejected"
		result["message"] = "All promises were rejected"
		return Reject(result)
	}

	return New(func(resolve, reject func(v any)) error {
		reasons := make([]any, len(proms))
		var count int32 = 0
		resolved := false

		QueueMicrotask(func() {
			for i, prom := range proms {
				go func(index int, p Promise) {
					state, value := Wait(p)

					if resolved {
						return
					}

					if state == Fulfilled {
						resolved = true
						resolve(value)
						return
					}

					reasons[index] = value

					if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) && !resolved {
						result := make(map[string]any)
						result["errors"] = reasons
						result["stack"] = "AggregateError: All promises were rejected"
						result["message"] = "All promises were rejected"
						reject(result)
					}
				}(i, prom)
			}
		})

		return nil
	})
}

/*
Race 等待第一个 Promise 完成。
  - 新 Promise 会在第一个 Promise 完成后解决或拒绝，解决值或拒绝理由跟随第一个完成的 Promise。
*/
func Race(proms []Promise) Promise {
	if proms == nil {
		return Reject("TypeError: nil is not iterable")
	}

	return New(func(resolve, reject func(v any)) error {
		if len(proms) == 0 {
			return nil
		}

		settled := false

		QueueMicrotask(func() {
			for _, prom := range proms {
				go func(p Promise) {
					state, value := Wait(p)

					if settled {
						return
					}

					settled = true
					if state == Fulfilled {
						resolve(value)
					} else {
						reject(value)
					}
				}(prom)
			}
		})

		return nil
	})
}

/*
Resolve 返回一个已解决的 Promise，解决值为指定值 value。

如果 value 已经是 Promise，则直接返回该 Promise。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/resolve
*/
func Resolve(value any) Promise {
	if prom, ok := value.(Promise); ok {
		return prom
	}

	return New(func(resolve, reject func(v any)) error {
		resolve(value)
		return nil
	})
}

/*
Reject 返回一个已拒绝的 Promise，拒绝理由为指定值。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/reject
*/
func Reject(reason any) Promise {
	return New(func(resolve, reject func(v any)) error {
		reject(reason)
		return nil
	})
}

/*
Try 接受一个任意类型的回调函数（无论其是同步或异步，返回结果或抛出异常），并将其结果封装成一个 Promise。
  - fn 任意类型的回调函数，接受任意数量的参数，函数返回值格式与 [ThenCallback] 相同。
  - args 将要传递给 fn 函数的参数列表。

返回一个 Promise，其状态可以是：
  - 已解决（Fulfilled）：如果 fn 函数返回一个非错误值。
  - 已拒绝（Rejected）：如果 fn 函数返回一个错误值。
  - 异步解决或拒绝：如果 fn 函数返回一个 Promise，新 Promise 会吸收该 Promise 的状态。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/try
*/
func Try(fn func(...any) (any, error), args ...any) Promise {
	return New(func(resolve, reject func(v any)) error {
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

/*
PromiseWithResolvers 创建一个新的 Promise 实例，同时返回 resolve 和 reject 函数，
对应于传入给 Promise() 构造函数执行器的两个参数。

这使得可以在 Promise 外部手动解决或拒绝 Promise。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/withResolvers#%E6%8F%8F%E8%BF%B0
*/
func PromiseWithResolvers() (Promise, func(any), func(any)) {
	var resolve, reject func(any)
	p := New(func(res func(any), rej func(any)) error {
		resolve = res
		reject = rej
		return nil
	})
	return p, resolve, reject
}

/*
QueueMicrotask 将回调函数添加到微任务队列末尾。
*/
func QueueMicrotask(fn func()) {
	microtaskQuene <- fn
	resetLoopTimer()
}

// ====== 辅助函数 ======
func resolve(prom *promiseImpl, value any) {
	if prom == nil {
		return
	}

	if prom.state != Pending {
		return
	}

	if value == prom {
		// 2.3.1 如果 Promise 和已决值相同，则抛出 TypeError 异常
		reject(prom, "TypeError: Chaining cycle detected for promise")
		return
	}

	// 2.3.2
	if x, ok := value.(Promise); ok {
		// 2.3.2 如果已决值是 Promise 对象，则采用其状态
		QueueMicrotask(func() {
			x.Then(func(v any) (any, error) {
				resolve(prom, v)
				return nil, nil
			}, func(r any) (any, error) {
				reject(prom, r)
				return nil, nil
			})
		})
		return
	}
	// 2.3.3 同上

	// 2.3.4 其他情况，则使用 value 作为已决值
	prom.state = Fulfilled
	prom.result = value
	close(prom.settled)
	QueueMicrotask(func() {
		flushHandlers(prom)
	})
}

func reject(prom *promiseImpl, reason any) {
	if prom == nil {
		return
	}

	if prom.state != Pending {
		return
	}

	prom.state = Rejected
	prom.result = reason
	close(prom.settled)
	QueueMicrotask(func() {
		flushHandlers(prom)
	})
}

func queueTask(fn func()) {
	taskQuene <- fn
	resetLoopTimer()
}

func resetLoopTimer() {
	if !loopTimer.Stop() {
		select {
		case <-loopTimer.C:
		default:
		}
	}
	loopTimer.Reset(timeout + margin)
}

func flushHandlers(cur *promiseImpl) {
	// 2.2.6 then 可以注册多次，且会按照注册顺序执行
	for {
		select {
		case hdl := <-cur.settledHandlers:

			var res any
			var err error
			if cur.state == Fulfilled {
				if hdl.onFulfilled == nil {
					// 2.2.1 如果回调不是函数，则忽略（穿透->2.2.7.3）
					resolve(hdl.prom, cur.result)
					continue
				} else {
					// 2.2.2
					res, err = hdl.onFulfilled(cur.result)
					if err != nil {
						// 2.2.7.2 如果回调函数抛出一个异常 e，则新 Promise 实例必须被拒绝，且拒绝原因为 e
						reject(hdl.prom, res)
						continue
					}
				}
			} else {
				if hdl.onRejected == nil {
					// 2.2.1 如果回调不是函数，则忽略（穿透->2.2.7.4）
					reject(hdl.prom, cur.result)
					continue
				} else {
					// 2.2.3
					res, err = hdl.onRejected(cur.result)
					if err != nil {
						// 2.2.7.2 如果回调函数抛出一个异常 e，则新 Promise 实例必须被拒绝，且拒绝原因为 e
						reject(hdl.prom, res)
						continue
					}
				}
			}

			// 2.2.7.1 如果回调函数返回一个值，则使用该值来 resolve 新 Promise
			resolve(hdl.prom, res)
		default:
			return
		}
	}
}

// ====== 辅助函数【结束】 ======

/*
Wait 同步阻塞等待 Promise 完成并返回其状态和值。

注意：
  - 此函数会阻塞当前 goroutine，直到 Promise 完成。
  - Promise 可能永远不会完成，谨慎使用。
*/
func Wait(prom Promise) (state string, value any) {
	<-prom.Done()
	return prom.State(), prom.Result()
}

/*
GetCloseFn 返回一个关闭函数，调用后会关闭事件循环。

获取此函数后，必须调用此函数来结束事件循环，否则将不会停止。
*/
func GetCloseFn() func() {
	granted.Do(func() {
		timeout = magic
		originTimeout = magic
		resetLoopTimer()

		go func() {
			for {
				select {
				case <-time.After(time.Hour * 24):
					resetLoopTimer()
				case <-done:
					return
				}
			}
		}()
	})
	return closeFn
}

/*
startEventLoop 模拟事件循环：清空微队列 -> 执行一个宏任务（如有） -> 清空微队列 ...
*/
func startEventLoop() {
loop:
	for {
		select {
		case mtask := <-microtaskQuene:
			resetLoopTimer()
			mtask()
			resetLoopTimer()
		default:
			select {
			case task := <-taskQuene:
				resetLoopTimer()
				task()
				resetLoopTimer()
			default:
				select {
				case mtask := <-microtaskQuene:
					resetLoopTimer()
					mtask()
					resetLoopTimer()
				case task := <-taskQuene:
					resetLoopTimer()
					task()
					resetLoopTimer()
				case <-done:
					break loop
				case <-loopTimer.C:
					break loop
				}
			}
		}
	}

	close(schedulerDone)
}
