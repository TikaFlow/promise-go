package promise

import (
	"time"

	pool "github.com/TikaFlow/worker-pool"
)

// NewPromise [EventLoop.NewPromise]
func (el *eventLoopImpl) NewPromise(exec Executor) Promise {
	if exec == nil {
		panic("Promise executor must be a function")
	}

	prom := &promiseImpl{
		value:           nil,
		reason:          nil,
		state:           Pending,
		settledHandlers: make(chan *handler, 128),
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

	if err := exec(res, rej); err != nil {
		rej(err)
	}
	return prom
}

// QueueMicrotask [EventLoop.QueueMicrotask]
func (el *eventLoopImpl) QueueMicrotask(fn func()) {
	if fn == nil {
		return
	}
	el.microtaskQueue <- fn
}

// 执行所有任务队列中的任务
func (el *eventLoopImpl) flushTasks() {
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

// 将一个异步任务添加到工作池
func (el *eventLoopImpl) pushTask(fn func()) {
	el.worker.Add(fn)
}

// 事件循环：清空微队列 -> 执行一个宏任务（如有） -> 清空微队列 ...
func (el *eventLoopImpl) run() {
	for {
		select {
		case task := <-el.microtaskQueue:
			task()
		case <-el.done:
			return
		default:
			select {
			case task := <-el.macrotaskQueue:
				task()
			case <-el.done:
				return
			default:
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
	}
}

// EventLoop 核心接口，定义了 [Promise] 异步相关的API
type EventLoop interface {
	// Stop 停止事件循环，将会等待其管理的异步任务完成才会返回
	Stop()

	// SetTimeout 模拟 setTimeout 函数，在指定毫秒数后调用回调函数
	//   - callback 回调函数
	//   - millis 延迟执行的毫秒数，自动修正负值的延迟为0
	//
	// # return
	//
	// 定时器 ID，可通过调用 [EventLoop.ClearTimeout] 函数来清除定时器
	SetTimeout(callback func(), millis int64) int

	// SetInterval 模拟 setInterval 函数，以指定毫秒数为周期，重复调用回调函数
	//   - callback 回调函数
	//   - millis 延迟执行的毫秒数，自动修正负值的延迟为0
	//
	// # return
	//
	// 定时器 ID，可通过调用 [EventLoop.ClearInterval] 函数来清除定时器
	SetInterval(callback func(), millis int64) int

	// ClearTimeout 清除由 [EventLoop.SetTimeout] 函数创建的定时器
	//   - id 定时器ID
	ClearTimeout(id int)

	// ClearInterval 清除由 [EventLoop.SetInterval] 函数创建的定时器
	//   - id 定时器ID
	ClearInterval(id int)

	// NewPromise 创建一个新的 [Promise] 实例
	//   - exec 执行器函数，用于定义 [Promise] 的异步操作
	NewPromise(exec Executor) Promise

	// All 等待所有输入解决
	//   - 如果 inputs 的所有元素都成功解决，新 [Promise] 也会成功解决，且解决值为一个包含所有元素解决值的数组
	//   - 如果任何一个元素被拒绝，新 [Promise] 也会被拒绝，且拒绝理由为第一个被拒绝的元素的拒绝理由
	All(inputs ...any) Promise

	// AllSettled 等待所有 [Promise] 已决（解决或拒绝）
	//   - 新 [Promise] 会在所有 [Promise] 已决后解决，解决值为一个包含所有 [Promise] 完成状态和结果的数组
	AllSettled(inputs ...any) Promise

	// Any 等待 inputs 中第一个成功解决的元素
	//   - 如果任何一个 [Promise] 解决，新 [Promise] 也会被解决，且解决值为第一个被解决的解决值
	//   - 如果所有 [Promise] 都被拒绝，新 [Promise] 也会被拒绝，且拒绝理由为 [AggregateError]，
	//     其包含所有 [Promise] 拒绝理由的数组，顺序为 [Promise] 数组中的顺序
	Any(inputs ...any) Promise

	// Async 将 fn 作为一个异步任务执行
	//
	// 类似于 `go fn()`，但会在一个专用的 worker-pool 中进行，且能获取返回值
	//
	// [todo] [fixme] 返回值、异常，应修复
	// 返回一个 [Promise] 实例，并在 fn 函数执行完成后变为解决状态，解决值是 fn 的返回值
	// 若 fn 函数抛出异常 err，则 [Promise] 实例会被拒绝，且拒绝理由为 err
	Async(fn func()) Promise

	// Await 等待 Promise 完成并获取解决值，可设定超时时间，以免无限等待
	//   - prom 需要等待的 [Promise] 实例，如果不是 [Promise] 实例，则会被直接返回
	//   - timeout 超时时间，单位为毫秒
	//
	// # return
	//
	//   - v 目标 prom 的解决值
	//   - err 拒绝理由，当 err 存在时，代表 [Promise] 被拒绝，此时 v 的值无意义
	Await(prom any, timeout int64) (v any, err error)

	// Delay 返回一个新的 [Promise]，其状态会在延迟时间后被解决
	//   - prom 将会使用的解决值，如果 prom 是 [Promise] 实例，则会等待其已决后才开始计时；
	//     如果是一个已拒绝的 [Promise]，则会立即拒绝新 [Promise]
	//   - millis 延迟时间，单位为毫秒
	Delay(prom any, millis int64) Promise

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
	Each(it func(item any, index int, arrLen int) any, inputs ...any) Promise

	// Filter 过滤数组中的元素
	//
	// # return
	//
	// 一个新的 [Promise]，其状态将会是：
	//   - 已解决（[Fulfilled]）：如果所有 [Promise] 都成功解决，解决值是过滤后的数组
	//   - 已拒绝（[Rejected]）：如果任何一个 [Promise] 被拒绝
	Filter(filter func(item any) bool, inputs ...any) Promise

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
	Map(mapper func(item any) any, inputs ...any) Promise

	// On 绑定一个钩子函数
	//   - event 钩子事件类型，可选值为 [ [OnCreated] | [OnChained] | [OnFulfilled] | [OnRejected] | [OnSettled] ]
	//   - hook 钩子函数，当事件触发时调用，并以触发该事件的 [Promise] 实例作为参数
	//
	// # return
	//
	// 绑定成功返回钩子函数的唯一标识，可用于后续解绑钩子函数，失败返回空字符串
	On(event HookType, hook func(p Promise)) string

	// Off 解绑一个钩子函数
	//   - event 钩子事件类型，可选值为 [ [OnCreated] | [OnChained] | [OnFulfilled] | [OnRejected] | [OnSettled] ]
	//   - key 要解绑的钩子函数的唯一标识，由 [EventLoop.On] 方法返回
	//
	// event 与 key 必须匹配，否则将解绑失败
	//
	// [todo] 返回bool以表明解绑是否成功
	Off(event HookType, key string)

	// PromiseWithResolvers 创建一个新的 [Promise] 实例，同时返回 resolve 和 reject 函数，
	// 对应于传入给 Promise() 构造函数执行器的两个参数
	//
	// 这使得可以在 [Promise] 外部手动解决或拒绝 [Promise]，详见 [MDN]
	//
	// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/withResolvers#%E6%8F%8F%E8%BF%B0
	PromiseWithResolvers() (Promise, func(any), func(any))

	// QueueMicrotask 将回调函数添加到微任务队列末尾
	QueueMicrotask(fn func())

	// Race 等待第一个 [Promise] 已决，
	// 新 [Promise] 会在第一个 [Promise] 已决后解决或拒绝，解决值或拒绝理由跟随第一个完成的 [Promise]
	Race(inputs ...any) Promise

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
	Reduce(reducer func(acc any, cur any) any, initial any, inputs ...any) Promise

	// Reject 返回一个已拒绝的 [Promise]，拒绝理由为指定值 reason，详见 [MDN]
	//
	// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/reject
	Reject(reason any) Promise

	// Resolve 返回一个已解决的 [Promise]，解决值为指定值 value
	//
	// 如果 value 已经是 [Promise]，则直接返回该 [Promise]，详见 [MDN]
	//
	// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/resolve
	Resolve(input any) Promise

	// Some 等待 inputs 中前 num 个元素解决
	//   - 如果 num 个元素解决，新 [Promise] 也会被解决，且解决值为一个包含所有元素解决值的数组，
	//     其顺序为被解决的顺序
	//   - 如果太多元素被拒绝，以至于新 [Promise] 永远无法满足，那么新 [Promise] 会立即被拒绝，
	//     且拒绝理由为 [AggregateError]，其包含所有元素拒绝理由的数组，顺序为被拒绝的顺序
	//
	// 注意与 [EventLoop.Any] 的不同，不仅是解决值的格式不同，拒绝理由的顺序也不同
	Some(num int, inputs ...any) Promise

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
	Try(fn func(...any) (any, error), args ...any) Promise
}

// 事件循环的内部实现
type eventLoopImpl struct {
	EventLoop
	microtaskQueue chan func()
	macrotaskQueue chan func()
	looper         pool.Pool
	scheduler      pool.Pool
	worker         pool.Pool
	timeline       *timeLine
	hooks          *promiseHooks
	done           chan struct{}
}

// StartEventLoop 启动一个事件循环
// 事件循环会持续运行，直到调用 [EventLoop.Stop] 方法关闭
//   - workerCount: 工作线程数量
func StartEventLoop(workerCount int) EventLoop {
	el := &eventLoopImpl{
		microtaskQueue: make(chan func(), 1024*10),
		macrotaskQueue: make(chan func(), 1024*10),
		done:           make(chan struct{}),
	}

	config := &pool.Config{
		BufferSize: 1024,
	}
	el.looper = pool.New(1, config)
	el.scheduler = pool.New(1, config)
	el.worker = pool.New(workerCount, nil)
	el.timeline = &timeLine{
		nextID:    0,
		tasks:     make([]*timedTask, 0, 1024*10),
		timer:     time.NewTimer(100 * 365 * 24 * time.Hour),
		taskCh:    make(chan *timedTask, 1024*10),
		clearCh:   make(chan int, 1024*10),
		eventLoop: el,
	}
	el.hooks = &promiseHooks{
		createdHookKeys:   make([]string, 0, 64),
		chainedHookKeys:   make([]string, 0, 64),
		fulfilledHookKeys: make([]string, 0, 64),
		rejectedHookKeys:  make([]string, 0, 64),
		settledHookKeys:   make([]string, 0, 64),
		hooks:             make(map[string]func(p Promise)),
	}

	el.looper.Add(el.run)
	el.scheduler.Add(el.timeline.run)

	return el
}

// Stop [EventLoop.Stop]
func (el *eventLoopImpl) Stop() {
	el.flushTasks()
	close(el.done)
	_ = el.looper.Close()
	_ = el.scheduler.Close()
	_ = el.worker.Close()
}
