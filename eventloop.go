package promise

import (
	"sync/atomic"

	pool "github.com/TikaFlow/worker-pool"
)

// EventLoop 核心接口，定义了 [Promise] 异步相关的API
//
// # 通过 nil 调用 EventLoop 的任何方法都可能触发 panic
type EventLoop struct {
	microtaskQueue chan func()
	macrotaskQueue chan func()
	looper         pool.Pool
	scheduler      pool.Pool
	worker         pool.Pool
	timeline       *timeLine
	hooks          *promiseHooks
	done           chan struct{}
}

// 执行所有任务队列中的任务
func (el *EventLoop) flushTasks() {
	for _, task := range el.timeline.tasks {
		el.timeline.queueMacrotask(task.callback)
	}
	close(el.macrotaskQueue)
	close(el.microtaskQueue)
	for task := range el.microtaskQueue {
		task()
	}
	for task := range el.macrotaskQueue {
		task()
	}
}

// 事件循环：清空微队列 -> 执行一个宏任务（如有） -> 清空微队列 ...
func (el *EventLoop) run() {
	for {
		select {
		case task := <-el.microtaskQueue:
			task()
			continue
		case <-el.done:
			return
		default:
		}

		select {
		case task := <-el.macrotaskQueue:
			task()
			continue
		case <-el.done:
			return
		default:
		}

		select {
		case task := <-el.microtaskQueue:
			task()
		case task := <-el.macrotaskQueue:
			task()
		case <-el.done:
			return
		}
	}
}

// 将一个异步任务添加到工作池
func (el *EventLoop) pushTask(fn func()) {
	el.worker.Add(fn)
}

// All 等待所有输入解决
//   - 如果 inputs 的所有元素都成功解决，新 [Promise] 也会成功解决，且解决值为一个包含所有元素解决值的数组
//   - 如果任何一个元素被拒绝，新 [Promise] 也会被拒绝，且拒绝理由为第一个被拒绝的元素的拒绝理由
func (el *EventLoop) All(inputs ...any) *Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}
	if len(inputs) == 0 {
		return el.Resolve(make([]any, 0))
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		results := make([]any, len(inputs))
		var count int32 = 0

		for index, item := range inputs {
			prom := el.Resolve(item)
			prom.Then(func(v any) (any, error) {
				results[index] = v
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(inputs) {
					resolve(results)
				}
				return nil, nil
			}, func(reason error) (any, error) {
				reject(reason)
				return nil, nil
			})
		}

		return nil
	})
}

// AllSettled 等待所有 [Promise] 已决（解决或拒绝）
//   - 新 [Promise] 会在所有 [Promise] 已决后解决，解决值为一个包含所有 [Promise] 完成状态和结果的数组
func (el *EventLoop) AllSettled(inputs ...any) *Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}
	if len(inputs) == 0 {
		return el.Resolve(make([]map[string]any, 0))
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		type result struct {
			Status string
			Value  any
			Reason any
		}

		length := len(inputs)
		results := make([]result, length)
		var count int32 = 0
		for index, item := range inputs {
			prom := el.Resolve(item)
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
			prom.Then(func(v any) (any, error) {
				results[index] = result{Status: Fulfilled, Value: v}
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == length {
					settleData()
				}
				return nil, nil
			}, func(reason error) (any, error) {
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

// Any 等待 inputs 中第一个解决的元素
//   - 如果任何一个 [Promise] 解决，新 [Promise] 也会被解决，且解决值为第一个被解决的解决值
//   - 如果所有 [Promise] 都被拒绝，新 [Promise] 也会被拒绝，且拒绝理由为 [AggregateError]，
//     其包含所有 [Promise] 拒绝理由的数组，顺序为 [Promise] 数组中的顺序
func (el *EventLoop) Any(inputs ...any) *Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}
	if len(inputs) == 0 {
		err := NewAggregateError(make([]error, 0), "All promises were rejected", "All promises were rejected")
		return el.Reject(err)
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		length := len(inputs)
		reasons := make([]error, length)

		var count int32 = 0

		for index, item := range inputs {
			prom := el.Resolve(item)
			prom.Then(func(v any) (any, error) {
				resolve(v)
				return nil, nil
			}, func(reason error) (any, error) {
				reasons[index] = reason
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == length {
					reject(NewAggregateError(reasons, "All promises were rejected", "All promises were rejected"))
				}
				return nil, nil
			})
		}

		return nil
	})
}

// Async 将 fn 作为一个异步任务执行
//
// 类似于 `go fn()`，但会在一个专用的 worker-pool 中进行，且能获取返回值
//
// # return
//
// 一个新的 [Promise] 实例，并在 fn 函数执行完成后变为解决状态，解决值是 fn 的返回值
// 若 fn 函数抛出异常 err，则 [Promise] 实例会被拒绝，且拒绝理由为 err
func (el *EventLoop) Async(fn func() (any, error)) *Promise {
	if fn == nil {
		return el.Reject(NewTypeError("fn must be a function"))
	}

	return el.NewPromise(func(resolve, reject func(any)) error {
		task := func() {
			// fn 内 panic 同样视为拒绝理由
			defer func() {
				if r := recover(); r != nil {
					reject(r)
				}
			}()
			v, err := fn()
			if err != nil {
				reject(err)
				return
			}
			resolve(v)
		}
		el.pushTask(task)
		return nil
	})
}

// Await 等待 Promise 已决并获取解决值，可设定超时时间，以免无限等待
//   - prom 需要等待的 [Promise] 实例，如果不是 [Promise] 实例，则会被直接返回
//   - timeout 超时时间，单位为毫秒
//
// # return
//
//   - v 目标 prom 的解决值
//   - err 拒绝理由，当 err 存在时，代表 [Promise] 被拒绝，此时 v 的值无意义
func (el *EventLoop) Await(prom any, timeout int64) (v any, err error) {
	if timeout <= 0 {
		return nil, NewRangeError("await timeout must be greater than 0")
	}

	prom2, ok := prom.(*Promise)
	if !ok {
		return prom, nil
	}

	wait := el.NewPromise(func(resolve, reject func(v any)) error {
		el.SetTimeout(func() {
			reject(NewTimeoutError("await timeout"))
		}, timeout)
		return nil
	})

	select {
	case <-prom2.Done():
		if prom2.State() == Rejected {
			err = prom2.Reason()
		} else {
			v = prom2.Value()
		}
	case <-wait.Done():
		if wait.State() == Rejected {
			err = wait.Reason()
		} else {
			v = wait.Value()
		}
	}

	return
}

// ClearInterval 清除由 [EventLoop.SetInterval] 函数创建的定时器
//   - id 定时器ID
func (el *EventLoop) ClearInterval(id int) {
	if id == -1 {
		return
	}
	el.timeline.clearCh <- id
}

// ClearTimeout 清除由 [EventLoop.SetTimeout] 函数创建的定时器
//   - id 定时器ID
func (el *EventLoop) ClearTimeout(id int) {
	if id == -1 {
		return
	}
	el.timeline.clearCh <- id
}

// Delay 返回一个新的 [Promise]，其状态会在延迟时间后被解决
//   - prom 将会使用的解决值，如果 prom 是 [Promise] 实例，则会等待其已决后才开始计时；
//     如果是一个已拒绝的 [Promise]，则会立即拒绝新 [Promise]
//   - millis 延迟时间，单位为毫秒
func (el *EventLoop) Delay(prom any, millis int64) *Promise {
	return el.NewPromise(func(resolve, reject func(v any)) error {
		el.Resolve(prom).Then(func(v2 any) (any, error) {
			el.SetTimeout(func() {
				resolve(v2)
			}, millis)
			return nil, nil
		}, func(r error) (any, error) {
			reject(r)
			return nil, nil
		})
		return nil
	})
}

// Timeout 返回一个新的 [Promise]：
//   - 若 prom 在 millis 毫秒内未 settled，则以 *TimeoutError 拒绝新 [Promise]；
//   - 否则跟随 prom 的状态（同值 / 同理由）
//
// millis 负值按 0 处理（与 [EventLoop.SetTimeout] 一致）。它是框架层的显式超时组合子：
// 按需为某个 promise 配置超时，不影响其他 promise，也不改动 [Promise] 结构。
func (el *EventLoop) Timeout(prom any, millis int64) *Promise {
	base := el.Resolve(prom)
	return el.NewPromise(func(resolve, reject func(v any)) error {
		id := el.SetTimeout(func() {
			reject(NewTimeoutError("promise timed out"))
		}, millis)

		base.Then(func(v any) (any, error) {
			el.ClearTimeout(id)
			resolve(v)
			return nil, nil
		}, func(r error) (any, error) {
			el.ClearTimeout(id)
			reject(r)
			return nil, nil
		})
		return nil
	})
}

// Each 按顺序等待 inputs 的每个元素已决，每个元素的结果会被传递给迭代器 it
// 如果 it 返回一个 [Promise]，则会等待该 [Promise] 完成后再继续迭代；
// 如果当前迭代对象是 [Promise]，则会等待其完成后再继续迭代；
// 迭代过程中遇到任何一个已拒绝 [Promise]，新 Promise 也会以同样的理由被拒绝
//   - it 对每个元素进行操作的函数，接受三个参数：item（当前元素）、index（当前元素的索引）、arrLen（数组长度）
//   - inputs 需要迭代的输入
//
// 由于迭代器的输出会被丢弃，因此适合副作用操作，如打印日志等
//
// # return
//
// 一个 [Promise]，其状态将会是：
//   - 已解决（[Fulfilled]）：如果所有迭代都成功解决，解决值是包含原始输入解决值的数组
//   - 已拒绝（[Rejected]）：如果迭代过程中任何一个 [Promise] 被拒绝
func (el *EventLoop) Each(it func(item any, index int, arrLen int) any, inputs ...any) *Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}
	if len(inputs) == 0 {
		return el.Resolve(make([]any, 0))
	}
	if it == nil {
		return el.Reject(NewTypeError("nil is not a function"))
	}

	prom := el.Resolve("start")
	arrLen := len(inputs)
	result := make([]any, arrLen)
	for index, item := range inputs {
		prom = prom.
			Then(func(any) (any, error) {
				return item, nil
			}, nil).
			Then(func(v any) (any, error) {
				result[index] = v
				return it(v, index, arrLen), nil
			}, nil)
	}

	return prom.
		Then(func(any) (any, error) {
			return result, nil
		}, nil)
}

// Filter 过滤数组中的元素
//
// # return
//
// 一个新的 [Promise]，其状态将会是：
//   - 已解决（[Fulfilled]）：如果所有 [Promise] 都成功解决，解决值是过滤后的数组
//   - 已拒绝（[Rejected]）：如果任何一个 [Promise] 被拒绝
func (el *EventLoop) Filter(filter func(item any) bool, inputs ...any) *Promise {
	return el.Map(func(item any) any {
		return el.All(item, filter(item))
	}, inputs...).
		Then(func(v any) (any, error) {
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

// Map 对输入数组中的每个元素应用一个函数，返回一个新的 [Promise] 数组，
// 新数组的每个元素都是原数组对应元素应用函数后的结果
//   - mapper 对每个元素进行映射操作的函数，接受一个参数 item 并返回一个新值
//   - inputs 将被 mapper 处理的输入
//
// # return
//
// 一个 [Promise]，其状态将会是：
//   - 已解决（[Fulfilled]）：如果所有 [Promise] 都成功解决，且每个 [Promise] 的解决值都被 mapper 处理后得到新值。
//   - 已拒绝（[Rejected]）：如果任何一个 [Promise] 被拒绝
func (el *EventLoop) Map(mapper func(item any) any, inputs ...any) *Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}
	if len(inputs) == 0 {
		return el.Resolve(make([]any, 0))
	}
	if mapper == nil {
		return el.Reject(NewTypeError("nil is not a function"))
	}

	result := make([]any, len(inputs))
	for index, item := range inputs {
		result[index] = el.Resolve(item).Then(func(v any) (any, error) {
			return mapper(v), nil
		}, nil)
	}

	return el.All(result...)
}

// NewPromise 创建一个新的 [Promise] 实例
//   - exec 执行器函数，用于定义 [Promise] 的异步操作
func (el *EventLoop) NewPromise(exec Executor) *Promise {
	if exec == nil {
		panic("Promise executor must be a function")
	}

	prom := &Promise{
		value:           nil,
		reason:          nil,
		state:           Pending,
		settledHandlers: make([]*handler, 0),
		settled:         make(chan struct{}),
		eventLoop:       el,
	}
	el.hooks.callHooks(OnCreated, prom)

	res := func(data any) {
		prom.resolved.Do(func() {
			resolvePromise(prom, data)
		})
	}
	rej := func(reason any) {
		prom.resolved.Do(func() {
			rejectPromise(prom, reason)
		})
	}

	// 执行器内部发生 panic 时，同样按照规范将其作为拒绝理由处理。
	// 注：若执行器先 resolve 再 panic，sync.Once 保证该 panic 被忽略（规范 2.3.3 已决后忽略）。
	func() {
		defer func() {
			if r := recover(); r != nil {
				rej(r)
			}
		}()
		if err := exec(res, rej); err != nil {
			rej(err)
		}
	}()
	return prom
}

// Off 解绑一个钩子函数
//   - event 钩子事件类型，可选值为 [ [OnCreated] | [OnChained] | [OnFulfilled] | [OnRejected] | [OnSettled] ]
//   - key 要解绑的钩子函数的唯一标识，由 [EventLoop.On] 方法返回
//
// event 与 key 必须匹配，否则将解绑失败
//
// # return
//
// 表明解绑是否成功的 bool 值
func (el *EventLoop) Off(event HookType, key string) bool {
	if key == "" {
		return false
	}

	el.hooks.hooksLock.Lock()
	defer el.hooks.hooksLock.Unlock()

	var targetSlice *[]string

	switch event {
	case OnCreated:
		targetSlice = &el.hooks.createdHookKeys
	case OnChained:
		targetSlice = &el.hooks.chainedHookKeys
	case OnFulfilled:
		targetSlice = &el.hooks.fulfilledHookKeys
	case OnRejected:
		targetSlice = &el.hooks.rejectedHookKeys
	case OnSettled:
		targetSlice = &el.hooks.settledHookKeys
	}

	if targetSlice == nil {
		return false
	}
	if deleteFromSlice(targetSlice, key) {
		delete(el.hooks.hooks, key)
		return true
	}

	return false
}

// On 绑定一个钩子函数
//   - event 钩子事件类型，可选值为 [ [OnCreated] | [OnChained] | [OnFulfilled] | [OnRejected] | [OnSettled] ]
//   - hook 钩子函数，当事件触发时调用，并以触发该事件的 [Promise] 实例作为参数
//
// # return
//
// 绑定成功返回钩子函数的唯一标识，可用于后续解绑钩子函数，失败返回空字符串
func (el *EventLoop) On(event HookType, hook func(p *Promise)) string {
	if hook == nil {
		return ""
	}

	el.hooks.hooksLock.Lock()
	defer el.hooks.hooksLock.Unlock()

	var exist = true
	var key string
	for exist {
		key = string(event) + randString(16)
		_, exist = el.hooks.hooks[key]
	}

	switch event {
	case OnCreated:
		el.hooks.createdHookKeys = append(el.hooks.createdHookKeys, key)
		el.hooks.hooks[key] = hook
	case OnChained:
		el.hooks.chainedHookKeys = append(el.hooks.chainedHookKeys, key)
		el.hooks.hooks[key] = hook
	case OnFulfilled:
		el.hooks.fulfilledHookKeys = append(el.hooks.fulfilledHookKeys, key)
		el.hooks.hooks[key] = hook
	case OnRejected:
		el.hooks.rejectedHookKeys = append(el.hooks.rejectedHookKeys, key)
		el.hooks.hooks[key] = hook
	case OnSettled:
		el.hooks.settledHookKeys = append(el.hooks.settledHookKeys, key)
		el.hooks.hooks[key] = hook
	}

	return key
}

// PromiseWithResolvers 创建一个新的 [Promise] 实例，同时返回 resolve 和 reject 函数，
// 对应于传入给 Promise() 构造函数执行器的两个参数
//
// 这使得可以在 [Promise] 外部手动解决或拒绝 [Promise]，详见 [MDN]
//
// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/withResolvers#%E6%8F%8F%E8%BF%B0
func (el *EventLoop) PromiseWithResolvers() (*Promise, func(any), func(any)) {
	var resolve, reject func(any)
	p := el.NewPromise(func(res, rej func(v any)) error {
		resolve = res
		reject = rej
		return nil
	})
	return p, resolve, reject
}

// QueueMicrotask 将回调函数添加到微任务队列末尾
func (el *EventLoop) QueueMicrotask(fn func()) {
	if fn == nil {
		return
	}
	el.microtaskQueue <- fn
}

// Race 等待第一个 [Promise] 已决，
// 新 [Promise] 会在第一个 [Promise] 已决后解决或拒绝，解决值或拒绝理由跟随第一个完成的 [Promise]
func (el *EventLoop) Race(inputs ...any) *Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		if len(inputs) == 0 {
			return nil
		}

		for _, item := range inputs {
			prom := el.Resolve(item)
			prom.Then(func(v any) (any, error) {
				resolve(v)
				return nil, nil
			}, func(reason error) (any, error) {
				reject(reason)
				return nil, nil
			})
		}

		return nil
	})
}

// Reduce 对数组中的每个元素应用一个函数 reducer，将其结果累积到 acc 中，最后返回 acc 的值
//   - reducer 对每个元素进行操作的函数，接受两个参数 acc 和 cur，返回新的 acc
//   - initial 初始值
//   - inputs 被操作的数组
//
// # return
//
// 一个新的 [Promise]，其状态将会是：
//   - 已解决（[Fulfilled]）：如果所有 [Promise] 都成功解决，且每个 [Promise] 的解决值都被 reducer 处理后得到新值
//   - 已拒绝（[Rejected]）：如果任何一个 [Promise] 被拒绝
//
// 特殊情况：
//   - 如果 inputs 为空数组，直接返回初始值 initial
//   - 如果 initial 为 nil，且 inputs 只有一个元素，直接返回该元素
func (el *EventLoop) Reduce(reducer func(acc any, cur any) any, initial any, inputs ...any) *Promise {
	init := el.Resolve(initial)

	if len(inputs) == 0 {
		return init
	}

	return init.
		Then(func(v any) (any, error) {
			if v == nil && len(inputs) == 1 {
				return inputs[0], nil
			}

			acc := v
			result := el.Each(func(item any, index int, arrLen int) any {
				acc = reducer(acc, item)
				return nil
			}, inputs...).
				Then(func(v any) (any, error) {
					return acc, nil
				}, nil)
			return result, nil
		}, nil)
}

// Reject 返回一个已拒绝的 [Promise]，拒绝理由为指定值 reason，详见 [MDN]
//
// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/reject
func (el *EventLoop) Reject(reason any) *Promise {
	return el.NewPromise(func(resolve, reject func(v any)) error {
		reject(reason)
		return nil
	})
}

// Resolve 返回一个已解决的 [Promise]，解决值为指定值 value
//
// 如果 value 已经是 [Promise]，则直接返回该 [Promise]，详见 [MDN]
//
// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/resolve
func (el *EventLoop) Resolve(value any) *Promise {
	if prom, ok := value.(*Promise); ok {
		return prom
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		resolve(value)
		return nil
	})
}

// SetInterval 模拟 setInterval 函数，以指定毫秒数为周期，重复调用回调函数
//   - callback 回调函数
//   - millis 延迟执行的毫秒数，自动修正负值的延迟为0
//
// # return
//
// 定时器 ID，可通过调用 [EventLoop.ClearInterval] 函数来清除定时器
func (el *EventLoop) SetInterval(callback func(), millis int64) int {
	if callback == nil {
		return -1
	}
	return el.timeline.produceTask(callback, millis, true)
}

// SetTimeout 模拟 setTimeout 函数，在指定毫秒数后调用回调函数
//   - callback 回调函数
//   - millis 延迟执行的毫秒数，自动修正负值的延迟为0
//
// # return
//
// 定时器 ID，可通过调用 [EventLoop.ClearTimeout] 函数来清除定时器
func (el *EventLoop) SetTimeout(callback func(), millis int64) int {
	if callback == nil {
		return -1
	}
	return el.timeline.produceTask(callback, millis, false)
}

// Some 等待 inputs 中前 num 个元素解决
//   - 如果 num 个元素解决，新 [Promise] 也会被解决，且解决值为一个包含所有元素解决值的数组，
//     其顺序为被解决的顺序
//   - 如果太多元素被拒绝，以至于新 [Promise] 永远无法满足，那么新 [Promise] 会立即被拒绝，
//     且拒绝理由为 [AggregateError]，其包含所有元素拒绝理由的数组，顺序为被拒绝的顺序
//
// 注意与 [EventLoop.Any] 的不同，不仅是解决值的格式不同，拒绝理由的顺序也不同
func (el *EventLoop) Some(num int, inputs ...any) *Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}
	if num <= 0 {
		return el.Reject(NewRangeError("num must be greater than 0"))
	}
	if len(inputs) == 0 {
		err := NewAggregateError(make([]error, 0), "All promises were rejected", "All promises were rejected")
		return el.Reject(err)
	}
	if num > len(inputs) {
		return el.Reject(NewRangeError("no enough promises to resolve"))
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		threshold := len(inputs) - num + 1
		values := make([]any, 0, num)
		reasons := make([]error, 0, threshold)
		var resCount int32 = 0
		var rejCount int32 = 0

		for _, item := range inputs {
			prom := el.Resolve(item)
			prom.Then(func(v any) (any, error) {
				values = append(values, v)
				if newCount := atomic.AddInt32(&resCount, 1); int(newCount) == num {
					resolve(values)
				}
				return nil, nil
			}, func(reason error) (any, error) {
				reasons = append(reasons, reason)
				if newCount := atomic.AddInt32(&rejCount, 1); int(newCount) == threshold {
					result := NewAggregateError(reasons, "AggregateError: Too many promises were rejected", "Too many promises were rejected")
					reject(result)
				}
				return nil, nil
			})
		}
		return nil
	})
}

// Stop 停止事件循环，将会等待其管理的异步任务完成才会返回
func (el *EventLoop) Stop() {
	el.flushTasks()
	close(el.done)
	_ = el.looper.Close()
	_ = el.scheduler.Close()
	_ = el.worker.Close()
}

// Try 接受一个任意类型的回调函数，并将其结果封装成一个 [Promise]，详见 [MDN]。
//   - fn 任意类型的回调函数，接受任意数量的参数，函数返回值格式为 (any, error)
//   - args 将要传递给 fn 函数的参数列表
//
// # return
//
// 一个新的 [Promise]，其状态将会是：
//   - 已解决（[Fulfilled]）：如果 fn 函数返回一个普通值
//   - 已拒绝（[Rejected]）：如果 fn 函数返回了 err
//   - 异步解决或拒绝：如果 fn 函数返回一个 [Promise]，新 [Promise] 会吸收该 [Promise] 的状态
//
// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/try
func (el *EventLoop) Try(fn func(...any) (any, error), args ...any) *Promise {
	return el.NewPromise(func(resolve, reject func(v any)) error {
		if fn == nil {
			return NewTypeError("Promise executor must be a function")
		}

		result, err := fn(args...)
		if err != nil {
			reject(err)
			return nil
		}
		resolve(result)
		return nil
	})
}
