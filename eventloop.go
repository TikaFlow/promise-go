package promise

import (
	"sync"

	pool "github.com/TikaFlow/worker-pool"
)

// taskQueueBufSize 微任务/宏任务 Queue 的 pop 通道缓冲容量
const taskQueueBufSize = 1024

// EventLoop 核心接口，定义了 [Promise] 异步相关的API
//
// # 通过 nil 调用 EventLoop 的任何方法都可能触发 panic
type EventLoop struct {
	microtaskQueue *Queue[func()]
	macrotaskQueue *Queue[func()]
	looper         pool.Pool
	scheduler      pool.Pool
	worker         pool.Pool
	timeline       *timeLine
	hooks          *hooks
	done           chan struct{}
	stopOnce       sync.Once
}

// 执行所有任务队列中的任务
// 遍历 el.timeline.tasks 是无锁读，因为此时所有调度都停止，可安全访问
func (el *EventLoop) flushTasks() {
	for task := range el.microtaskQueue.Pop() {
		task()
	}
	for task := range el.macrotaskQueue.Pop() {
		task()
	}
	for _, task := range el.timeline.tasks {
		task.callback()
	}
}

// drainMicro 同步排空微任务队列中所有已入队的微任务。
//
// pop 通道为空但内部链表尚有元素（feed 尚未搬运）时，阻塞等待 feed 送达，
// 确保"先微后宏"的时序不被 feed 的异步搬运延迟破坏。
// 返回 false 表示事件循环应结束（pop 通道已关闭或收到 done 信号）。
func (el *EventLoop) drainMicro() bool {
	for {
		select {
		case task, ok := <-el.microtaskQueue.Pop():
			if !ok {
				return false
			}
			el.hooks.safeCall(PromisePanic, task)
			continue
		default:
		}
		// 所有即时可读的任务已消费完；若队列真为空则排空完成
		if el.microtaskQueue.empty() {
			return true
		}
		// 链表尚有数据在搬运途中，阻塞等待 feed 送达
		// 同时监听 done 防止事件循环关闭时死锁
		select {
		case task, ok := <-el.microtaskQueue.Pop():
			if !ok {
				return false
			}
			el.hooks.safeCall(PromisePanic, task)
		case <-el.done:
			return false
		}
	}
}

// 事件循环：清空微队列 -> 执行一个宏任务（如有） -> 清空微队列 ...
func (el *EventLoop) run() {
	for {
		if !el.drainMicro() {
			return
		}

		select {
		case task, ok := <-el.macrotaskQueue.Pop():
			if !ok {
				return
			}
			el.hooks.safeCall(TimerPanic, task)
			continue
		case <-el.done:
			return
		default:
		}

		select {
		case task, ok := <-el.microtaskQueue.Pop():
			if !ok {
				return
			}
			el.hooks.safeCall(PromisePanic, task)
		case task, ok := <-el.macrotaskQueue.Pop():
			if !ok {
				return
			}
			el.hooks.safeCall(TimerPanic, task)
		case <-el.done:
			return
		}
	}
}

// 将一个异步任务添加到工作池
func (el *EventLoop) pushTask(fn func()) {
	el.worker.Add(fn)
}

// OffPromise 解绑一个钩子函数
//   - event 钩子事件类型，可选值为 [ [PromiseCreated] | [PromiseChained] | [PromiseFulfilled] | [PromiseRejected] | [PromiseSettled] ]
//   - key 要解绑的钩子函数的唯一标识，由 [EventLoop.OnPromise] 方法返回
//
// event 与 key 必须匹配，否则将解绑失败
//
// # return
//
// 表明解绑是否成功的 bool 值
func (el *EventLoop) OffPromise(event HookType, key string) bool {
	if key == "" {
		return false
	}

	el.hooks.promiseHooksLock.Lock()
	defer el.hooks.promiseHooksLock.Unlock()

	var targetSlice *[]string

	switch event {
	case PromiseCreated:
		targetSlice = &el.hooks.promiseCreatedHookKeys
	case PromiseChained:
		targetSlice = &el.hooks.promiseChainedHookKeys
	case PromiseFulfilled:
		targetSlice = &el.hooks.promiseFulfilledHookKeys
	case PromiseRejected:
		targetSlice = &el.hooks.promiseRejectedHookKeys
	case PromiseSettled:
		targetSlice = &el.hooks.promiseSettledHookKeys
	}

	if targetSlice == nil {
		return false
	}
	if deleteFromSlice(targetSlice, key) {
		delete(el.hooks.promiseHooks, key)
		return true
	}

	return false
}

// OnPromise 绑定一个钩子函数
//   - event 钩子事件类型，可选值为 [ [PromiseCreated] | [PromiseChained] | [PromiseFulfilled] | [PromiseRejected] | [PromiseSettled] ]
//   - hook 钩子函数，当事件触发时调用，并以触发该事件的 [Promise] 实例作为参数
//
// # return
//
// 绑定成功返回钩子函数的唯一标识，可用于后续解绑钩子函数，失败返回空字符串
func (el *EventLoop) OnPromise(event HookType, hook func(p *Promise)) string {
	if hook == nil {
		return ""
	}

	el.hooks.promiseHooksLock.Lock()
	defer el.hooks.promiseHooksLock.Unlock()

	var exist = true
	var key string
	for exist {
		key = string(event) + randString(16)
		_, exist = el.hooks.promiseHooks[key]
	}

	switch event {
	case PromiseCreated:
		el.hooks.promiseCreatedHookKeys = append(el.hooks.promiseCreatedHookKeys, key)
		el.hooks.promiseHooks[key] = hook
	case PromiseChained:
		el.hooks.promiseChainedHookKeys = append(el.hooks.promiseChainedHookKeys, key)
		el.hooks.promiseHooks[key] = hook
	case PromiseFulfilled:
		el.hooks.promiseFulfilledHookKeys = append(el.hooks.promiseFulfilledHookKeys, key)
		el.hooks.promiseHooks[key] = hook
	case PromiseRejected:
		el.hooks.promiseRejectedHookKeys = append(el.hooks.promiseRejectedHookKeys, key)
		el.hooks.promiseHooks[key] = hook
	case PromiseSettled:
		el.hooks.promiseSettledHookKeys = append(el.hooks.promiseSettledHookKeys, key)
		el.hooks.promiseHooks[key] = hook
	}

	return key
}

// Stop 停止事件循环，将会等待其管理的异步任务完成才会返回
//
// 注意：Stop 会阻塞等待 looper 关闭，因此禁止在事件循环内（如 Then/定时器回调中）调用
func (el *EventLoop) Stop() {
	el.stopOnce.Do(func() {
		close(el.done)
		el.microtaskQueue.Close()
		el.macrotaskQueue.Close()
		_ = el.worker.Close()
		_ = el.scheduler.Close()
		_ = el.looper.Close()
		el.flushTasks()
	})
}

// OffPanic 解绑一个 panic 钩子函数
//   - event 钩子事件类型，可选值为 [ [AllPanic] | [PromisePanic] | [AsyncPanic] | [ExecutorPanic] | [HookPanic] | [TimerPanic] ]
//   - key 要解绑的钩子函数的唯一标识，由 [EventLoop.OnPanic] 方法返回
//
// event 与 key 必须匹配，否则将解绑失败
//
// # return
//
// 表明解绑是否成功的 bool 值
func (el *EventLoop) OffPanic(event HookType, key string) bool {
	if key == "" {
		return false
	}

	el.hooks.panicHooksLock.Lock()
	defer el.hooks.panicHooksLock.Unlock()

	var targetSlice *[]string
	switch event {
	case AllPanic:
		targetSlice = &el.hooks.allPanicHookKeys
	case PromisePanic:
		targetSlice = &el.hooks.promisePanicHookKeys
	case AsyncPanic:
		targetSlice = &el.hooks.asyncPanicHookKeys
	case ExecutorPanic:
		targetSlice = &el.hooks.executorPanicHookKeys
	case HookPanic:
		targetSlice = &el.hooks.hookPanicHookKeys
	case TimerPanic:
		targetSlice = &el.hooks.timerPanicHookKeys
	}

	if targetSlice == nil {
		return false
	}
	if deleteFromSlice(targetSlice, key) {
		delete(el.hooks.panicHooks, key)
		return true
	}

	return false
}

// OnPanic 绑定一个 panic 钩子函数
//   - event 钩子事件类型，可选值为 [ [AllPanic] | [PromisePanic] | [AsyncPanic] | [ExecutorPanic] | [HookPanic] | [TimerPanic] ]
//   - hook 钩子函数，当对应 panic 发生时调用，接收 panic 值
//
// # return
//
// 绑定成功返回钩子函数的唯一标识，可用于后续解绑钩子函数，失败返回空字符串
func (el *EventLoop) OnPanic(event HookType, hook func(r any)) string {
	if hook == nil {
		return ""
	}

	el.hooks.panicHooksLock.Lock()
	defer el.hooks.panicHooksLock.Unlock()

	var exist = true
	var key string
	for exist {
		key = string(event) + randString(16)
		_, exist = el.hooks.panicHooks[key]
	}

	switch event {
	case AllPanic:
		el.hooks.allPanicHookKeys = append(el.hooks.allPanicHookKeys, key)
		el.hooks.panicHooks[key] = hook
	case PromisePanic:
		el.hooks.promisePanicHookKeys = append(el.hooks.promisePanicHookKeys, key)
		el.hooks.panicHooks[key] = hook
	case AsyncPanic:
		el.hooks.asyncPanicHookKeys = append(el.hooks.asyncPanicHookKeys, key)
		el.hooks.panicHooks[key] = hook
	case ExecutorPanic:
		el.hooks.executorPanicHookKeys = append(el.hooks.executorPanicHookKeys, key)
		el.hooks.panicHooks[key] = hook
	case HookPanic:
		el.hooks.hookPanicHookKeys = append(el.hooks.hookPanicHookKeys, key)
		el.hooks.panicHooks[key] = hook
	case TimerPanic:
		el.hooks.timerPanicHookKeys = append(el.hooks.timerPanicHookKeys, key)
		el.hooks.panicHooks[key] = hook
	}

	return key
}
