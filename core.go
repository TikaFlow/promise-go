package promise

import (
	"io"
	"math"
	"time"

	pool "github.com/TikaFlow/worker-pool"
)

/*
NewPromise 创建一个新的 Promise 实例。
  - exec 执行器函数，用于定义 Promise 的异步操作。
*/
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
	el.hooks.callHooks(PromiseCreated, prom)

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

/*
QueueMicrotask 将回调函数添加到微任务队列末尾。
*/
func (el *eventLoopImpl) QueueMicrotask(fn func()) {
	el.microtaskQueue <- fn
}

// flushTasks 执行所有任务队列中的任务
func (el *eventLoopImpl) flushTasks() {
	for _, task := range el.timeline.tasks {
		el.timeline.queueMacrotask(task.callback)
	}
	close(el.macrotaskQueue)
	close(el.microtaskQueue)
	for task := range el.microtaskQueue {
		el.pushTask(task)
	}
	for task := range el.macrotaskQueue {
		el.pushTask(task)
	}
}

func (el *eventLoopImpl) pushTask(fn func()) {
	el.worker.Add(fn)
}

/*
eventLoop 模拟事件循环：清空微队列 -> 执行一个宏任务（如有） -> 清空微队列 ...
*/
func (el *eventLoopImpl) run() {
	for {
		select {
		case task := <-el.microtaskQueue:
			el.pushTask(task)
		case <-el.done:
			return
		default:
			select {
			case task := <-el.macrotaskQueue:
				el.pushTask(task)
			default:
				select {
				case task := <-el.microtaskQueue:
					el.pushTask(task)
				case task := <-el.macrotaskQueue:
					el.pushTask(task)
				case <-el.done:
					return
				}
			}
		}
	}
}

type EventLoop interface {
	io.Closer
	SetTimeout(callback func(), millis int64) int
	SetInterval(callback func(), millis int64) int
	ClearTimeout(id int)
	ClearInterval(id int)
	NewPromise(exec Executor) Promise
	All(inputs ...any) Promise
	AllSettled(inputs ...any) Promise
	Any(inputs ...any) Promise
	Async(fn func()) Promise
	Await(prom any, timeout int64) (v any, err error)
	Delay(prom any, millis int64) Promise
	Each(it func(item any, index int, arrLen int) any, inputs ...any) Promise
	Filter(filter func(item any) bool, inputs ...any) Promise
	Map(mapper func(item any) any, inputs ...any) Promise
	On(event HookType, hook func(p Promise)) string
	Off(event HookType, key string)
	PromiseWithResolvers() (Promise, func(any), func(any))
	QueueMicrotask(fn func())
	Race(inputs ...any) Promise
	Reduce(reducer func(acc any, cur any) any, initial any, inputs ...any) Promise
	Reject(reason any) Promise
	Resolve(input any) Promise
	Some(num int, inputs ...any) Promise
	Try(fn func(...any) (any, error), args ...any) Promise
}

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

// StartEventLoop 启动一个新的事件循环。
// 事件循环会持续运行，直到调用 Close 方法关闭。
// 参数：
// - workerCount: 工作线程数量
func StartEventLoop(workerCount int) EventLoop {
	config := &pool.Config{
		BufferSize: 1024,
	}
	mgr := pool.New(1, config)
	scheduler := pool.New(1, config)
	worker := pool.New(workerCount, nil)
	timeline := &timeLine{
		tasks:   make([]*timedTask, 0, 1024*10),
		timer:   time.NewTimer(time.Duration(math.MaxInt64)),
		taskCh:  make(chan *timedTask, 1024*10),
		clearCh: make(chan int, 1024*10),
	}
	hooks := &promiseHooks{
		createdHookKeys:   make([]string, 0, 64),
		chainedHookKeys:   make([]string, 0, 64),
		fulfilledHookKeys: make([]string, 0, 64),
		rejectedHookKeys:  make([]string, 0, 64),
		settledHookKeys:   make([]string, 0, 64),
		hooks:             make(map[string]func(p Promise)),
	}

	el := &eventLoopImpl{
		microtaskQueue: make(chan func(), 1024*10),
		macrotaskQueue: make(chan func(), 1024*10),
		looper:         mgr,
		worker:         worker,
		timeline:       timeline,
		hooks:          hooks,
		done:           make(chan struct{}),
	}

	mgr.Add(el.run)
	scheduler.Add(timeline.run)

	return el
}

func StartClassicEventLoop() EventLoop {
	return StartEventLoop(1)
}

// Close 关闭事件循环。[io.Closer.Close]
func (el *eventLoopImpl) Close() error {
	el.flushTasks()
	close(el.done)
	e1 := el.looper.Close()
	e2 := el.worker.Close()
	if e1 != nil {
		return e1
	}
	if e2 != nil {
		return e2
	}
	return nil
}
