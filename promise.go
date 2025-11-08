package promise

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

// ====== 包变量 =======
var (
	microtaskQuene chan func()
	started        bool
)

func init() {
	// 有缓冲，避免阻塞
	microtaskQuene = make(chan func(), 1024)
}

// ====== 常量定义 ======
const (
	Pending   = "pending"
	Fulfilled = "fulfilled"
	Rejected  = "rejected"
)

// ====== 类型定义 ======
type callback func(any) (any, error)
type executor func(resolve, reject func(any)) error

// handler 表示待处理的 Promise 回调
type handler struct {
	onFulfilled callback
	onRejected  callback
	prom        *promiseImpl // 即将返回的 Promise 实例（新）
}

// Thenable 是一个定义了 then 方法的对象
type Thenable interface {
	State() string
	Result() any
	Done() chan struct{}
	Then(onFulfilled, onRejected callback) Promise
}

// Promise 是一个拥有 then 方法的对象，其行为符合 Promise/A+ 规范
type Promise interface {
	Thenable
	Catch(onRejected callback) Promise
	Finally(onFinally func() (any, error)) Promise
}

// ====== 实现类 ======
type promiseImpl struct {
	Promise
	state           string
	value           any
	settledHandlers []handler
	done            chan struct{}
}

// 实现 Thenable 接口
func (prom *promiseImpl) State() string {
	return prom.state
}

func (prom *promiseImpl) Result() any {
	return prom.value
}

func (prom *promiseImpl) Done() chan struct{} {
	return prom.done
}

// 实现 Promise 接口
func (prom *promiseImpl) Then(onFulfilled callback, onRejected callback) Promise {
	prom2 := New(func(resolve, reject func(any)) error {
		return nil
	})
	prom.settledHandlers = append(prom.settledHandlers, handler{
		onFulfilled: onFulfilled,
		onRejected:  onRejected,
		prom:        prom2.(*promiseImpl),
	})
	microtaskQuene <- func() {
		flushHandlers(prom)
	}
	// 2.2.7 then 方法返回的新 Promise 实例的状态由回调函数的执行结果决定
	return prom2
}

// ref https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/catch
func (prom *promiseImpl) Catch(onRejected callback) Promise {
	return prom.Then(nil, onRejected)
}

func (prom *promiseImpl) Finally(onFinally func() (any, error)) Promise {
	cb := func(v any) (any, error) {
		// 默认穿透
		if onFinally == nil {
			return v, nil
		}

		res, err := onFinally()
		// 报错则拒绝
		if err != nil {
			return res, err
		}

		// 是一个拒绝的 Promise 实例，则以同样的理由拒绝
		if result, ok := res.(Thenable); ok {
			if result.State() == Rejected {
				reason := result.Result()
				return reason, errors.New("finally callback returns a rejected Promise")
			}
		}

		// 其他情况忽略 onFinally 的返回值
		return v, nil
	}
	return prom.Then(cb, cb)
}

// ****** 附加接口实现 ******
// 实现 String 方法
func (prom *promiseImpl) String() string {
	return fmt.Sprintf("Promise<%s>, result: %v", prom.state, prom.value)
}

// ====== 静态方法 ======
func New(exec executor) Promise {
	if exec == nil {
		panic("Promise executor must be a function")
	}

	prom := &promiseImpl{
		state:           Pending,
		value:           nil,
		settledHandlers: make([]handler, 0, 10),
		done:            make(chan struct{}),
	}

	res := func(data any) {
		resolve(prom, data)
	}
	rej := func(reason any) {
		reject(prom, reason)
	}

	if err := exec(res, rej); err != nil {
		rej(err)
	}
	return prom
}

func All(proms []Promise) Promise {
	return New(func(resolve, reject func(any)) error {
		// 处理空数组情况
		if len(proms) == 0 {
			resolve(make([]any, 0))
			return nil
		}

		results := make([]any, len(proms))
		var count int32 = 0
		// 使用闭包变量来防止多次resolve/reject
		resolved := false
		rejected := false

		// 将异步处理逻辑放入微队列
		microtaskQuene <- func() {
			// 遍历所有Promise并处理它们的结果
			for i, prom := range proms {
				go func(index int, p Promise) {
					state, value := Wait(p)

					// 检查是否已经有结果
					if rejected || resolved {
						return
					}

					if state == Rejected {
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
		}

		return nil
	})
}

func AllSettled(proms []Promise) Promise {
	return New(func(resolve, reject func(any)) error {
		// 处理空数组情况
		if len(proms) == 0 {
			resolve(make([]map[string]any, 0))
			return nil
		}

		// 定义结果结构，包含status和value/reason
		type result struct {
			Status string
			Value  any
			Reason any
		}

		results := make([]result, len(proms))
		var count int32 = 0
		// 将异步处理逻辑放入微队列
		microtaskQuene <- func() {
			// 遍历所有Promise并处理它们的结果
			for i, prom := range proms {
				go func(index int, p Promise) {
					state, value := Wait(p)

					// 无论成功失败都记录结果
					if state == Fulfilled {
						results[index] = result{Status: Fulfilled, Value: value}
					} else {
						results[index] = result{Status: Rejected, Reason: value}
					}

					// 使用原子操作增加计数器 - 原子累加确保在并发环境中计数准确，避免竞态条件
					if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) {
						// 将result结构转换为map格式
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
		}

		return nil
	})
}

func Any(proms []Promise) Promise {
	return New(func(resolve, reject func(any)) error {
		// 处理空数组情况
		if len(proms) == 0 {
			result := make(map[string]any)
			result["errors"] = make([]any, 0)
			result["stack"] = "AggregateError: All promises were rejected"
			result["message"] = "All promises were rejected"
			reject(result)
			return nil
		}

		reasons := make([]any, len(proms))
		var count int32 = 0
		// 使用闭包变量来防止多次resolve/reject
		resolved := false
		rejected := false

		// 将异步处理逻辑放入微队列
		microtaskQuene <- func() {
			// 遍历所有Promise并处理它们的结果
			for i, prom := range proms {
				go func(index int, p Promise) {
					state, value := Wait(p)

					// 检查是否已经有结果
					if rejected || resolved {
						return
					}

					if state == Fulfilled {
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
		}

		return nil
	})
}

func Race(proms []Promise) Promise {
	return New(func(resolve, reject func(any)) error {
		// 处理空数组情况
		if len(proms) == 0 {
			// 空数组永远不会解决或拒绝，返回一个pending状态的Promise
			return nil
		}

		// 使用闭包变量来防止多次resolve/reject
		settled := false

		// 将异步处理逻辑放入微队列
		microtaskQuene <- func() {
			// 遍历所有Promise，任何一个完成就决定新Promise的状态
			for _, prom := range proms {
				go func(p Promise) {
					// 等待Promise完成
					state, value := Wait(p)

					// 检查是否已经有结果
					if settled {
						return
					}

					// 根据第一个完成的Promise状态决定新Promise状态
					settled = true
					if state == Fulfilled {
						resolve(value)
					} else {
						reject(value)
					}
				}(prom)
			}
		}

		return nil
	})
}

// ref: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/resolve
func Resolve(value any) Promise {
	// 如果值已经是Promise，直接返回
	if prom, ok := value.(Promise); ok {
		return prom
	}

	// 创建一个已解决的Promise
	return New(func(resolve, reject func(any)) error {
		resolve(value)
		return nil
	})
}

// ref: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/reject
func Reject(reason any) Promise {
	// 创建一个已拒绝的Promise
	return New(func(resolve, reject func(any)) error {
		reject(reason)
		return nil
	})
}

func Try(fn func(...any) (any, error), args ...any) Promise {
	return New(func(res func(any), rej func(any)) error {
		if fn == nil {
			rej("Promise executor must be a function")
			return nil
		}

		result, err := fn(args...)
		if err != nil {
			rej(result)
			return err
		}
		res(result)
		return nil
	})
}

// ref: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/withResolvers#%E6%8F%8F%E8%BF%B0
func PromiseWithResolvers() (Promise, func(any), func(any)) {
	var resolve, reject func(any)
	p := New(func(res func(any), rej func(any)) error {
		resolve = res
		reject = rej
		return nil
	})
	return p, resolve, reject
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
	if x, ok := value.(Thenable); ok {
		// 2.3.2 如果已决值是 Promise 对象，则采用其状态
		microtaskQuene <- func() {
			x.Then(func(v any) (any, error) {
				resolve(prom, v)
				return nil, nil
			}, func(r any) (any, error) {
				reject(prom, r)
				return nil, nil
			})
		}
		return
	}
	// 2.3.3 同上

	// 2.3.4 其他情况，则使用 value 作为已决值
	prom.state = Fulfilled
	prom.value = value
	close(prom.done)
	microtaskQuene <- func() {
		flushHandlers(prom)
	}
}

func reject(prom *promiseImpl, reason any) {
	if prom == nil {
		return
	}

	if prom.state != Pending {
		return
	}

	prom.state = Rejected
	prom.value = reason
	close(prom.done)
	microtaskQuene <- func() {
		flushHandlers(prom)
	}
}

func flushHandlers(cur *promiseImpl) {
	if cur.state == Pending {
		return
	}

	if len(cur.settledHandlers) == 0 {
		return
	}

	// 2.2.6 then 可以注册多次，且会按照注册顺序执行
	microtaskQuene <- func() {
		for len(cur.settledHandlers) > 0 {
			hdl := cur.settledHandlers[0]

			cur.settledHandlers = cur.settledHandlers[1:]

			var res any
			var err error
			if cur.state == Fulfilled {
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
		}
	}
}

// ====== 其他方法 ======
// 同步阻塞等待 Promise 完成并返回其状态和值
func Wait(prom Thenable) (state string, value any) {
	<-prom.Done()
	return prom.State(), prom.Result()
}

// 模拟事件循环
func Play(timeout time.Duration) {
	if started {
		return
	}
	started = true
	for {
		select {
		case fn := <-microtaskQuene:
			fn()
		case <-time.After(timeout):
			// 超时退出
			close(microtaskQuene)
			return
		}
	}
}
