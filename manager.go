package promise

import (
	"sync/atomic"
	"time"

	"github.com/TikaFlow/promise-go/ipromise"
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
)

func init() {
	go scheduleTask()
	go startEventLoop()
}

func closeFn() {
	close(done)
}

// New 创建一个新的 Promise 实例。
// executor 执行器函数，用于定义 Promise 的异步操作。
// 返回一个新的 Promise 实例。
func New(exec ipromise.Executor) ipromise.Promise {
	if exec == nil {
		panic("Promise executor must be a function")
	}

	prom := &promiseImpl{
		state:           ipromise.Pending,
		value:           nil,
		settledHandlers: make(chan *handler, 128),
		done:            make(chan struct{}),
	}

	called := false
	res := func(data any) {
		if called {
			return
		}
		called = true
		resolve(prom, data)
	}
	rej := func(reason any) {
		if called {
			return
		}
		called = true
		reject(prom, reason)
	}

	if err := exec(res, rej); err != nil {
		rej(err)
	}
	return prom
}

// All 等待所有 Promise 解决。
// 如果所有 Promise 都成功解决，新 Promise 也会成功解决，且解决值为一个包含所有 Promise 解决值的数组；
// 如果任何一个 Promise 被拒绝，新 Promise 也会被拒绝，且拒绝理由为第一个被拒绝的 Promise 的拒绝理由。
func All(proms []ipromise.Promise) ipromise.Promise {
	if proms == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(proms) == 0 {
		return Resolve(make([]any, 0))
	}

	return New(func(resolve, reject func(v any)) error {
		results := make([]any, len(proms))
		var count int32 = 0
		resolved := false
		rejected := false

		QueueMicrotask(func() {
			for i, prom := range proms {
				go func(index int, p ipromise.Promise) {
					state, value := Wait(p)

					if rejected || resolved {
						return
					}

					if state == ipromise.Rejected {
						rejected = true
						reject(value)
						return
					}

					results[index] = value

					if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) && !rejected {
						resolved = true
						resolve(results)
					}
				}(i, prom)
			}
		})

		return nil
	})
}

// AllSettled 等待所有 Promise 完成（无论成功失败）。
// 新 Promise 会在所有 Promise 完成后解决，解决值为一个包含所有 Promise 完成状态和结果的数组。
func AllSettled(proms []ipromise.Promise) ipromise.Promise {
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
				go func(index int, p ipromise.Promise) {
					state, value := Wait(p)

					if state == ipromise.Fulfilled {
						results[index] = result{Status: ipromise.Fulfilled, Value: value}
					} else {
						results[index] = result{Status: ipromise.Rejected, Reason: value}
					}

					if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) {
						finalResults := make([]map[string]any, len(results))
						for i, r := range results {
							finalResults[i] = make(map[string]any)
							finalResults[i]["status"] = r.Status
							if r.Status == ipromise.Fulfilled {
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

// Any 等待第一个 Promise 解决。
// 如果任何一个 Promise 解决，新 Promise 也会被解决，且解决值为第一个被解决的 Promise 的解决值。
// 如果所有 Promise 都被拒绝，新 Promise 也会被拒绝，且拒绝理由为一个包含所有 Promise 拒绝理由的数组。
func Any(proms []ipromise.Promise) ipromise.Promise {
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
		rejected := false

		QueueMicrotask(func() {
			for i, prom := range proms {
				go func(index int, p ipromise.Promise) {
					state, value := Wait(p)

					if rejected || resolved {
						return
					}

					if state == ipromise.Fulfilled {
						resolved = true
						resolve(value)
						return
					}

					reasons[index] = value

					if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) && !resolved {
						// 所有Promise都被拒绝，拒绝新Promise
						rejected = true
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

// Race 等待第一个 Promise 完成。
// 新 Promise 会在第一个 Promise 完成后解决或拒绝，解决值或拒绝理由为第一个完成的 Promise 的解决值或拒绝理由。
func Race(proms []ipromise.Promise) ipromise.Promise {
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
				go func(p ipromise.Promise) {
					state, value := Wait(p)

					if settled {
						return
					}

					settled = true
					if state == ipromise.Fulfilled {
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

// Resolve 返回一个已解决的 Promise，解决值为指定值。
// 如果值已经是 Promise，则直接返回该 Promise。
// ref: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/resolve
func Resolve(value any) ipromise.Promise {
	// 如果值已经是Promise，直接返回
	if prom, ok := value.(ipromise.Promise); ok {
		return prom
	}

	return New(func(resolve, reject func(v any)) error {
		resolve(value)
		return nil
	})
}

// Reject 返回一个已拒绝的 Promise，拒绝理由为指定值。
// ref: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/reject
func Reject(reason any) ipromise.Promise {
	return New(func(resolve, reject func(v any)) error {
		reject(reason)
		return nil
	})
}

// Try 接受一个任意类型的回调函数（无论其是同步或异步，返回结果或抛出异常），并将其结果封装成一个 Promise。
func Try(fn func(...any) (any, error), args ...any) ipromise.Promise {
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

// PromiseWithResolvers 创建一个新的 Promise 实例，同时返回 resolve 和 reject 函数，
// 对应于传入给 Promise() 构造函数执行器的两个参数。
// 这使得可以在 Promise 外部手动解决或拒绝 Promise。
// ref: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/withResolvers#%E6%8F%8F%E8%BF%B0
func PromiseWithResolvers() (ipromise.Promise, func(any), func(any)) {
	var resolve, reject func(any)
	p := New(func(res func(any), rej func(any)) error {
		resolve = res
		reject = rej
		return nil
	})
	return p, resolve, reject
}

// QueueMicrotask 将回调函数添加到微任务队列末尾。
func QueueMicrotask(fn func()) {
	microtaskQuene <- fn
	resetLoopTimer()
}

// ====== 辅助函数 ======
func resolve(prom *promiseImpl, value any) {
	if prom == nil {
		return
	}

	if prom.state != ipromise.Pending {
		return
	}

	if value == prom {
		// 2.3.1 如果 Promise 和已决值相同，则抛出 TypeError 异常
		reject(prom, "TypeError: Chaining cycle detected for promise")
		return
	}

	// 2.3.2
	if x, ok := value.(ipromise.Promise); ok {
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
	prom.state = ipromise.Fulfilled
	prom.value = value
	close(prom.done)
	QueueMicrotask(func() {
		flushHandlers(prom)
	})
}

func reject(prom *promiseImpl, reason any) {
	if prom == nil {
		return
	}

	if prom.state != ipromise.Pending {
		return
	}

	prom.state = ipromise.Rejected
	prom.value = reason
	close(prom.done)
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
			if cur.state == ipromise.Fulfilled {
				if hdl.onFulfilled == nil {
					// 2.2.1 如果回调不是函数，则忽略（穿透->2.2.7.3）
					resolve(hdl.prom, cur.value)
					continue
				} else {
					// 2.2.2
					res, err = hdl.onFulfilled(cur.value)
					if err != nil {
						// 2.2.7.2 如果回调函数抛出一个异常 e，则新 Promise 实例必须被拒绝，且拒绝原因为 e
						reject(hdl.prom, res)
						continue
					}
				}
			} else {
				if hdl.onRejected == nil {
					// 2.2.1 如果回调不是函数，则忽略（穿透->2.2.7.4）
					reject(hdl.prom, cur.value)
					continue
				} else {
					// 2.2.3
					res, err = hdl.onRejected(cur.value)
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

// Wait 同步阻塞等待 Promise 完成并返回其状态和值。
func Wait(prom ipromise.Promise) (state string, value any) {
	<-prom.Done()
	return prom.State(), prom.Result()
}

// GetCloseFn 返回一个关闭函数，调用后会关闭事件循环。
// 获取此函数后，必须调用此函数来结束事件循环，否则将不会停止。
func GetCloseFn() func() {
	if timeout < magic {
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
	}
	return closeFn
}

// startEventLoop 模拟事件循环：清空微队列 -> 执行一个宏任务（如有） -> 清空微队列 ...
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
