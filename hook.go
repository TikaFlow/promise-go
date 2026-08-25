package promise

import "sync"

// HookType 钩子类型
type HookType string

// 以下常量用于 Promise 事件钩子，回调签名为 func(p *Promise)。
const (
	// PromiseCreated 当 Promise 实例被创建时调用。
	PromiseCreated HookType = "PROMISE_CREATED"
	// PromiseChained 当 Promise 实例被链式调用时调用。
	PromiseChained HookType = "PROMISE_CHAINED"
	// PromiseFulfilled 当 Promise 实例解决时调用。
	PromiseFulfilled HookType = "PROMISE_FULFILLED"
	// PromiseRejected 当 Promise 实例拒绝时调用。
	PromiseRejected HookType = "PROMISE_REJECTED"
	// PromiseSettled 当 Promise 实例被解决（无论成功或拒绝）时调用。
	PromiseSettled HookType = "PROMISE_SETTLED"
)

// 以下常量用于 Panic 事件钩子，回调签名为 func(r any)。
const (
	// TimerPanic 当定时器任务发生 panic 时调用。
	TimerPanic HookType = "TIMER_PANIC"
)

// 钩子实例定义
type hooks struct {
	promiseCreatedHookKeys   []string
	promiseChainedHookKeys   []string
	promiseFulfilledHookKeys []string
	promiseRejectedHookKeys  []string
	promiseSettledHookKeys   []string
	promiseHooks             map[string]func(p *Promise)
	promiseHooksLock         sync.RWMutex

	timerPanicHookKeys []string
	panicHooks         map[string]func(r any)
	panicHooksLock     sync.RWMutex
}

// callPromiseHooks 通过钩子类型调用 Promise 事件钩子。
//
// 钩子函数在锁外执行。因此钩子内可安全调用 [EventLoop.OnPromise] / [EventLoop.OffPromise]
// （二者需要写锁），不会因"持有读锁时尝试升级为写锁"而自死锁。
func (hk *hooks) callPromiseHooks(event HookType, p *Promise) {
	var keys []string
	switch event {
	case PromiseCreated:
		keys = hk.promiseCreatedHookKeys
	case PromiseChained:
		keys = hk.promiseChainedHookKeys
	case PromiseFulfilled:
		keys = hk.promiseFulfilledHookKeys
	case PromiseRejected:
		keys = hk.promiseRejectedHookKeys
	case PromiseSettled:
		keys = hk.promiseSettledHookKeys
	}
	callHooks(&hk.promiseHooksLock, hk.promiseHooks, keys, p)
}

// callPanicHooks 通过钩子类型调用 Panic 事件钩子，接收 panic 值。
//
// 钩子函数在锁外执行，且每个钩子以独立的 recover 包裹，防止钩子自身 panic
// 二次引发事件循环崩溃。
func (hk *hooks) callPanicHooks(event HookType, r any) {
	var keys []string
	switch event {
	case TimerPanic:
		keys = hk.timerPanicHookKeys
	}

	callHooks(&hk.panicHooksLock, hk.panicHooks, keys, r)
}

// callHooks 泛型辅助：在锁内收集函数快照，解锁后逐个执行。
//
// 适用于任何钩子族系（promise 钩子、panic 钩子等），
// 调用方只需提供对应的锁、映射表、key 切片和参数即可。
func callHooks[T any](lock *sync.RWMutex, hooks map[string]func(T), keys []string, arg T) {
	lock.RLock()
	fns := make([]func(T), 0, len(keys))
	for _, key := range keys {
		if fn, ok := hooks[key]; ok {
			fns = append(fns, fn)
		}
	}
	lock.RUnlock()

	for _, fn := range fns {
		fn(arg)
	}
}
