package promise

import (
	ip "github.com/TikaFlow/promise-go/ipromise"
	"sync/atomic"
	"time"
)

var (
	taskQuene      chan func()   = make(chan func(), 1024*10)
	microtaskQuene chan func()   = make(chan func(), 1024*10)
	timeout        time.Duration = time.Millisecond * 512
	margin         time.Duration = time.Millisecond * 128
	timer          *time.Timer   = time.NewTimer(timeout + margin)
	done           chan struct{} = make(chan struct{})
	magic          time.Duration = 86413*time.Second + margin
)

func init() {
	go startEventLoop()
}

func closeFn() {
	close(done)
}

// New 创建一个新的 Promise 实例。
func New(exec ip.Executor) ip.Promise {
	if exec == nil {
		panic("Promise executor must be a function")
	}

	prom := &promiseImpl{
		state:           ip.Pending,
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
func All(proms []ip.Promise) ip.Promise {
	if proms == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(proms) == 0 {
		return Resolve(make([]any, 0))
	}

	return New(func(resolve, reject func(v any)) error {
		results := make([]any, len(proms))
		var count int32 = 0
		// 使用闭包变量来防止多次resolve/reject
		resolved := false
		rejected := false

		// 将异步处理逻辑放入微队列
		QueueMicrotask(func() {
			// 遍历所有Promise并处理它们的结果
			for i, prom := range proms {
				go func(index int, p ip.Promise) {
					state, value := Wait(p)

					// 检查是否已经有结果
					if rejected || resolved {
						return
					}

					if state == ip.Rejected {
						// 如果有任何Promise被拒绝，立即拒绝新Promise
						rejected = true
						reject(value)
						return
					}

					// 存储已完成的Promise结果
					results[index] = value

					// 使用原子操作增加计数器 - 原子累加确保在并发环境中计数准确，避免竞态条件
					if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) && !rejected {
						// 所有Promise都已完成，resolve新Promise
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
func AllSettled(proms []ip.Promise) ip.Promise {
	if proms == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(proms) == 0 {
		return Resolve(make([]map[string]any, 0))
	}

	return New(func(resolve, reject func(v any)) error {
		// 定义结果结构，包含status和value/reason
		type result struct {
			Status string
			Value  any
			Reason any
		}

		results := make([]result, len(proms))
		var count int32 = 0
		// 将异步处理逻辑放入微队列
		QueueMicrotask(func() {
			// 遍历所有Promise并处理它们的结果
			for i, prom := range proms {
				go func(index int, p ip.Promise) {
					state, value := Wait(p)

					// 无论成功失败都记录结果
					if state == ip.Fulfilled {
						results[index] = result{Status: ip.Fulfilled, Value: value}
					} else {
						results[index] = result{Status: ip.Rejected, Reason: value}
					}

					// 使用原子操作增加计数器 - 原子累加确保在并发环境中计数准确，避免竞态条件
					if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) {
						// 将result结构转换为map格式
						finalResults := make([]map[string]any, len(results))
						for i, r := range results {
							finalResults[i] = make(map[string]any)
							finalResults[i]["status"] = r.Status
							if r.Status == ip.Fulfilled {
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
func Any(proms []ip.Promise) ip.Promise {
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
		// 使用闭包变量来防止多次resolve/reject
		resolved := false
		rejected := false

		// 将异步处理逻辑放入微队列
		QueueMicrotask(func() {
			// 遍历所有Promise并处理它们的结果
			for i, prom := range proms {
				go func(index int, p ip.Promise) {
					state, value := Wait(p)

					// 检查是否已经有结果
					if rejected || resolved {
						return
					}

					if state == ip.Fulfilled {
						// 如果有任何Promise被满足，立即resolve新Promise
						resolved = true
						resolve(value)
						return
					}

					// 记录拒绝原因
					reasons[index] = value

					// 使用原子操作增加计数器 - 原子累加确保在并发环境中计数准确，避免竞态条件
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
func Race(proms []ip.Promise) ip.Promise {
	if proms == nil {
		return Reject("TypeError: nil is not iterable")
	}

	return New(func(resolve, reject func(v any)) error {
		// 处理空数组情况
		if len(proms) == 0 {
			// 空数组永远不会解决或拒绝，返回一个pending状态的Promise
			return nil
		}

		// 使用闭包变量来防止多次resolve/reject
		settled := false

		// 将异步处理逻辑放入微队列
		QueueMicrotask(func() {
			// 遍历所有Promise，任何一个完成就决定新Promise的状态
			for _, prom := range proms {
				go func(p ip.Promise) {
					// 等待Promise完成
					state, value := Wait(p)

					// 检查是否已经有结果
					if settled {
						return
					}

					// 根据第一个完成的Promise状态决定新Promise状态
					settled = true
					if state == ip.Fulfilled {
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
func Resolve(value any) ip.Promise {
	// 如果值已经是Promise，直接返回
	if prom, ok := value.(ip.Promise); ok {
		return prom
	}

	// 创建一个已解决的Promise
	return New(func(resolve, reject func(v any)) error {
		resolve(value)
		return nil
	})
}

// Reject 返回一个已拒绝的 Promise，拒绝理由为指定值。
// ref: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/reject
func Reject(reason any) ip.Promise {
	// 创建一个已拒绝的Promise
	return New(func(resolve, reject func(v any)) error {
		reject(reason)
		return nil
	})
}

func Try(fn func(...any) (any, error), args ...any) ip.Promise {
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

// PromiseWithResolvers 创建一个新的 Promise 实例，同时返回 resolve 和 reject 函数。
// 这使得可以在 Promise 外部手动解决或拒绝 Promise。
// ref: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/withResolvers#%E6%8F%8F%E8%BF%B0
func PromiseWithResolvers() (ip.Promise, func(any), func(any)) {
	var resolve, reject func(any)
	p := New(func(res func(any), rej func(any)) error {
		resolve = res
		reject = rej
		return nil
	})
	return p, resolve, reject
}

// SetTimeout 模拟 setTimeout 函数，在指定毫秒数后调用回调函数。
// cb 回调函数
// millis 毫秒数
// 返回一个通道，可通过调用 ClearTimeout 函数来清除定时器。
func SetTimeout(cb func(), millis int64) chan struct{} {
	delay := abs(millis)

	originTimeout := timeout
	if delay > timeout.Milliseconds() {
		timeout = time.Duration(delay)*time.Millisecond + margin
		resetTimer()
	}

	ch := make(chan struct{})
	go func() {
		select {
		case <-ch:
			return
		case <-time.After(time.Duration(delay) * time.Millisecond):
			timeout = originTimeout
			queueTask(cb)
		}
	}()

	return ch
}

// SetInterval 模拟 setInterval 函数，在指定毫秒数后重复调用回调函数。
// cb 回调函数
// millis 毫秒数
// 返回一个通道，可通过调用 ClearInterval 函数来清除定时器。
func SetInterval(cb func(), millis int64) chan struct{} {
	delay := abs(millis)

	ch := make(chan struct{})
	var fn func()
	fn = func() {
		select {
		case <-ch:
			return
		default:
			cb()
			SetTimeout(fn, delay)
		}
	}

	tch := SetTimeout(fn, delay)
	go func() {
		select {
		case <-ch:
			close(tch)
		case <-time.After(time.Duration(delay) * time.Millisecond):
		}
	}()

	return ch
}

// ClearTimeout 清除由 SetTimeout 函数创建的定时器。
// ch 定时器通道
func ClearTimeout(ch chan struct{}) {
	close(ch)
}

// ClearInterval 清除由 SetInterval 函数创建的定时器。
// ch 定时器通道
func ClearInterval(ch chan struct{}) {
	close(ch)
}

// ====== 辅助函数 ======
func resolve(prom *promiseImpl, value any) {
	if prom == nil {
		return
	}

	if prom.state != ip.Pending {
		return
	}

	if value == prom {
		// 2.3.1 如果 Promise 和已决值相同，则抛出 TypeError 异常
		reject(prom, "TypeError: Chaining cycle detected for promise")
		return
	}

	// 2.3.2
	if x, ok := value.(ip.Promise); ok {
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
	prom.state = ip.Fulfilled
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

	if prom.state != ip.Pending {
		return
	}

	prom.state = ip.Rejected
	prom.value = reason
	close(prom.done)
	QueueMicrotask(func() {
		flushHandlers(prom)
	})
}

func queueTask(fn func()) {
	taskQuene <- fn
	resetTimer()
}

// QueueMicrotask 将回调函数添加到微任务队列末尾。
func QueueMicrotask(fn func()) {
	microtaskQuene <- fn
	resetTimer()
}

func resetTimer() {
	if timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(timeout)
}

func flushHandlers(cur *promiseImpl) {
	// 2.2.6 then 可以注册多次，且会按照注册顺序执行
	for {
		select {
		case hdl := <-cur.settledHandlers:

			var res any
			var err error
			if cur.state == ip.Fulfilled {
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

func abs(millis int64) int64 {
	switch {
	case millis >= 0:
		return millis
	case millis == (-1 << 63):
		return 1<<63 - 1
	default:
		return -millis
	}
}

// ====== 辅助函数【结束】 ======

// Wait 同步阻塞等待 Promise 完成并返回其状态和值。
func Wait(prom ip.Promise) (state string, value any) {
	<-prom.Done()
	return prom.State(), prom.Result()
}

func GetCloseFn() func() {
	if timeout != magic {
		// 第一次获取关闭函数
		timeout = magic
		resetTimer()
		// 无限重置 => 永不过期
		go func() {
			for {
				<-time.After(time.Hour * 24)
				resetTimer()
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
			resetTimer()
			mtask()
			resetTimer()
		default:
			select {
			case task := <-taskQuene:
				resetTimer()
				task()
				resetTimer()
			default:
				select {
				case mtask := <-microtaskQuene:
					resetTimer()
					mtask()
					resetTimer()
				case task := <-taskQuene:
					resetTimer()
					task()
					resetTimer()
				case <-done:
					break loop
				case <-timer.C:
					break loop
				}
			}
		}
	}
}
