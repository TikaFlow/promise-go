package promise

import (
	"errors"
	ip "github.com/TikaFlow/promise-go/ipromise"
	"sync/atomic"
	"time"
)

// PromiseManager 是一个 Promise 管理器，用于管理 Promise 的执行和状态变更。
// 仅限单线程使用，否则无法保证执行顺序。
type PromiseManager struct {
	microtaskQuene chan func()
	timer          *time.Timer
	timeout        time.Duration
	started        bool
}

// GetPromiseManager 获取一个 Promise 管理器实例。
func GetPromiseManager(timeout time.Duration) *PromiseManager {
	return &PromiseManager{
		microtaskQuene: make(chan func(), 1024),
		timer:          time.NewTimer(timeout),
		timeout:        timeout,
		started:        false,
	}
}

// New 创建一个新的 Promise 实例。
func (pm *PromiseManager) New(exec ip.Executor) ip.Promise {
	if exec == nil {
		panic("Promise executor must be a function")
	}

	prom := &promiseImpl{
		state:           ip.Pending,
		value:           nil,
		settledHandlers: make([]handler, 0, 10),
		done:            make(chan struct{}),
		manager:         pm,
	}

	called := false
	res := func(data any) {
		if called {
			return
		}
		called = true
		pm.resolve(prom, data)
	}
	rej := func(reason any) {
		if called {
			return
		}
		called = true
		pm.reject(prom, reason)
	}

	if err := exec(res, rej); err != nil {
		rej(err)
	}
	return prom
}

// All 等待所有 Promise 解决。
// 如果所有 Promise 都成功解决，新 Promise 也会成功解决，且解决值为一个包含所有 Promise 解决值的数组；
// 如果任何一个 Promise 被拒绝，新 Promise 也会被拒绝，且拒绝理由为第一个被拒绝的 Promise 的拒绝理由。
func (pm *PromiseManager) All(proms []ip.Promise) ip.Promise {
	if proms == nil {
		return pm.Reject("TypeError: nil is not iterable")
	}
	if len(proms) == 0 {
		return pm.Resolve(make([]any, 0))
	}

	return pm.New(func(resolve, reject func(v any)) error {
		results := make([]any, len(proms))
		var count int32 = 0
		// 使用闭包变量来防止多次resolve/reject
		resolved := false
		rejected := false

		// 将异步处理逻辑放入微队列
		pm.addTask(func() {
			// 遍历所有Promise并处理它们的结果
			for i, prom := range proms {
				go func(index int, p ip.Promise) {
					state, value := pm.Wait(p)

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
func (pm *PromiseManager) AllSettled(proms []ip.Promise) ip.Promise {
	if proms == nil {
		return pm.Reject("TypeError: nil is not iterable")
	}
	if len(proms) == 0 {
		return pm.Resolve(make([]map[string]any, 0))
	}

	return pm.New(func(resolve, reject func(v any)) error {
		// 定义结果结构，包含status和value/reason
		type result struct {
			Status string
			Value  any
			Reason any
		}

		results := make([]result, len(proms))
		var count int32 = 0
		// 将异步处理逻辑放入微队列
		pm.addTask(func() {
			// 遍历所有Promise并处理它们的结果
			for i, prom := range proms {
				go func(index int, p ip.Promise) {
					state, value := pm.Wait(p)

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
func (pm *PromiseManager) Any(proms []ip.Promise) ip.Promise {
	if proms == nil {
		return pm.Reject("TypeError: nil is not iterable")
	}
	if len(proms) == 0 {
		result := make(map[string]any)
		result["errors"] = make([]any, 0)
		result["stack"] = "AggregateError: All promises were rejected"
		result["message"] = "All promises were rejected"
		return pm.Reject(result)
	}

	return pm.New(func(resolve, reject func(v any)) error {
		reasons := make([]any, len(proms))
		var count int32 = 0
		// 使用闭包变量来防止多次resolve/reject
		resolved := false
		rejected := false

		// 将异步处理逻辑放入微队列
		pm.addTask(func() {
			// 遍历所有Promise并处理它们的结果
			for i, prom := range proms {
				go func(index int, p ip.Promise) {
					state, value := pm.Wait(p)

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
func (pm *PromiseManager) Race(proms []ip.Promise) ip.Promise {
	if proms == nil {
		return pm.Reject("TypeError: nil is not iterable")
	}

	return pm.New(func(resolve, reject func(v any)) error {
		// 处理空数组情况
		if len(proms) == 0 {
			// 空数组永远不会解决或拒绝，返回一个pending状态的Promise
			return nil
		}

		// 使用闭包变量来防止多次resolve/reject
		settled := false

		// 将异步处理逻辑放入微队列
		pm.addTask(func() {
			// 遍历所有Promise，任何一个完成就决定新Promise的状态
			for _, prom := range proms {
				go func(p ip.Promise) {
					// 等待Promise完成
					state, value := pm.Wait(p)

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
func (pm *PromiseManager) Resolve(value any) ip.Promise {
	// 如果值已经是Promise，直接返回
	if prom, ok := value.(ip.Promise); ok {
		return prom
	}

	// 创建一个已解决的Promise
	return pm.New(func(resolve, reject func(v any)) error {
		resolve(value)
		return nil
	})
}

// Reject 返回一个已拒绝的 Promise，拒绝理由为指定值。
// ref: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/reject
func (pm *PromiseManager) Reject(reason any) ip.Promise {
	// 创建一个已拒绝的Promise
	return pm.New(func(resolve, reject func(v any)) error {
		reject(reason)
		return nil
	})
}

func (pm *PromiseManager) Try(fn func(...any) (any, error), args ...any) ip.Promise {
	return pm.New(func(resolve, reject func(v any)) error {
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
func (pm *PromiseManager) PromiseWithResolvers() (ip.Promise, func(any), func(any)) {
	var resolve, reject func(any)
	p := pm.New(func(res func(any), rej func(any)) error {
		resolve = res
		reject = rej
		return nil
	})
	return p, resolve, reject
}

// ====== 辅助函数 ======
func (pm *PromiseManager) resolve(prom *promiseImpl, value any) {
	if prom == nil {
		return
	}

	if prom.state != ip.Pending {
		return
	}

	if value == prom {
		// 2.3.1 如果 Promise 和已决值相同，则抛出 TypeError 异常
		pm.reject(prom, "TypeError: Chaining cycle detected for promise")
		return
	}

	// 2.3.2
	if x, ok := value.(ip.Promise); ok {
		// 2.3.2 如果已决值是 Promise 对象，则采用其状态
		pm.addTask(func() {
			x.Then(func(v any) (any, error) {
				pm.resolve(prom, v)
				return nil, nil
			}, func(r any) (any, error) {
				pm.reject(prom, r)
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
	pm.addTask(func() {
		pm.flushHandlers(prom)
	})
}

func (pm *PromiseManager) reject(prom *promiseImpl, reason any) {
	if prom == nil {
		return
	}

	if prom.state != ip.Pending {
		return
	}

	prom.state = ip.Rejected
	prom.value = reason
	close(prom.done)
	pm.addTask(func() {
		pm.flushHandlers(prom)
	})
}

func (pm *PromiseManager) addTask(fn func()) {
	pm.microtaskQuene <- fn
	pm.resetTimer()
}

func (pm *PromiseManager) resetTimer() {
	if pm.timer.Stop() {
		select {
		case <-pm.timer.C:
		default:
		}
	}
	pm.timer.Reset(pm.timeout)
}

func (pm *PromiseManager) flushHandlers(cur *promiseImpl) {
	if cur.state == ip.Pending {
		return
	}

	if len(cur.settledHandlers) == 0 {
		return
	}

	// 2.2.6 then 可以注册多次，且会按照注册顺序执行
	pm.addTask(func() {
		for len(cur.settledHandlers) > 0 {
			hdl := cur.settledHandlers[0]

			cur.settledHandlers = cur.settledHandlers[1:]

			var res any
			var err error
			if cur.state == ip.Fulfilled {
				if hdl.onFulfilled == nil {
					// 2.2.1 如果回调不是函数，则忽略（穿透->2.2.7.3）
					pm.resolve(hdl.prom, cur.value)
					continue
				} else {
					// 2.2.2
					res, err = hdl.onFulfilled(cur.value)
					if err != nil {
						// 2.2.7.2 如果回调函数抛出一个异常 e，则新 Promise 实例必须被拒绝，且拒绝原因为 e
						pm.reject(hdl.prom, res)
						continue
					}
				}
			} else {
				if hdl.onRejected == nil {
					// 2.2.1 如果回调不是函数，则忽略（穿透->2.2.7.4）
					pm.reject(hdl.prom, cur.value)
					continue
				} else {
					// 2.2.3
					res, err = hdl.onRejected(cur.value)
					if err != nil {
						// 2.2.7.2 如果回调函数抛出一个异常 e，则新 Promise 实例必须被拒绝，且拒绝原因为 e
						pm.reject(hdl.prom, res)
						continue
					}
				}
			}

			// 2.2.7.1 如果回调函数返回一个值，则使用该值来 resolve 新 Promise
			pm.resolve(hdl.prom, res)
		}
	})
}

// ====== 辅助函数【结束】 ======

// Wait 同步阻塞等待 Promise 完成并返回其状态和值。
func (pm *PromiseManager) Wait(prom ip.Promise) (state string, value any) {
	<-prom.Done()
	return prom.State(), prom.Result()
}

// RunLoop 模拟事件循环，执行微任务队列中的任务。
// 如果 done 通道不为 nil，则在事件循环完成时关闭该通道，用于通知外部“事件循环已结束”。
// 如果多次调用，后续调用将报错“PromiseManager already started”。
// 注意：该方法将阻塞当前 goroutine，直到事件循环完成或超时退出；
// 如需异步运行事件循环或手动控制结束时间（而非默认超时时间），请使用 RunLoopAsync 方法。
func (pm *PromiseManager) RunLoop(done chan struct{}) error {
	if pm.started {
		return errors.New("PromiseManager already started")
	}
	pm.started = true
	for {
		select {
		case fn := <-pm.microtaskQuene:
			pm.resetTimer()
			fn()
		case <-pm.timer.C:
			// 超时退出
			close(pm.microtaskQuene)
			if done != nil {
				// 通知外部：事件循环已完成
				close(done)
			}
			return nil
		}
	}
}

// RunLoopAsync 模拟事件循环，异步执行微任务队列中的任务。
// 返回一个 done 通道，用于手动控制事件循环结束。
// 如果多次调用，后续调用将报错“PromiseManager already started”。
func (pm *PromiseManager) RunLoopAsync() (chan struct{}, error) {
	if pm.started {
		return nil, errors.New("PromiseManager already started")
	}
	pm.started = true

	done := make(chan struct{})
	go func() {
		for {
			select {
			case fn := <-pm.microtaskQuene:
				pm.resetTimer()
				fn()
			case <-done:
				// 接收到外部信号退出
				close(pm.microtaskQuene)
				return
			}
		}
	}()

	return done, nil
}
