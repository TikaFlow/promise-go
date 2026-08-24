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
	hooks          *promiseHooks
	done           chan struct{}
	stopOnce       sync.Once
}

// 执行所有任务队列中的任务
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
			task()
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
			task()
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
			task()
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
			task()
		case task, ok := <-el.macrotaskQueue.Pop():
			if !ok {
				return
			}
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
