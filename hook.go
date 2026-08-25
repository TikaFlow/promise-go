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
	// AllPanic 所有 panic 事件的通用钩子，任何 panic 都会先触发它。
	AllPanic HookType = "ALL_PANIC"
	// PromisePanic 当 Promise 回调发生 panic 时调用。
	PromisePanic HookType = "PROMISE_PANIC"
	// AsyncPanic 当 Async 任务发生 panic 时调用。
	AsyncPanic HookType = "ASYNC_PANIC"
	// HookPanic 当钩子函数自身发生 panic 时调用。
	HookPanic HookType = "HOOK_PANIC"
	// TimerPanic 当定时器任务发生 panic 时调用。
	TimerPanic HookType = "TIMER_PANIC"
)

// ignorePanic 内部专用事件：safeCall 捕获 panic 后静默吞掉，不再触发任何钩子。
// 用于钩子自身执行的包裹，避免钩子 panic 递归触发 HookPanic 造成啸叫。
const ignorePanic HookType = "IGNORE_PANIC"

// 钩子实例定义
type hooks struct {
	promiseCreatedHookKeys   []string
	promiseChainedHookKeys   []string
	promiseFulfilledHookKeys []string
	promiseRejectedHookKeys  []string
	promiseSettledHookKeys   []string
	promiseHooks             map[string]func(p *Promise)
	promiseHooksLock         sync.RWMutex

	allPanicHookKeys     []string
	promisePanicHookKeys []string
	asyncPanicHookKeys   []string
	hookPanicHookKeys    []string
	timerPanicHookKeys   []string
	panicHooks           map[string]func(r any)
	panicHooksLock       sync.RWMutex
}

// safeCall 以指定事件包裹 fn 的调用：fn 发生 panic 时捕获，先触发 AllPanic
// 再触发 event 对应的钩子，并返回 panic 值（无 panic 返回 nil）。
// event 为 ignorePanic 时静默吞掉 panic，不触发任何钩子。
func (hk *hooks) safeCall(event HookType, fn func()) (r any) {
	defer func() {
		if r = recover(); r != nil && event != ignorePanic {
			hk.callPanicHooks(AllPanic, r)
			hk.callPanicHooks(event, r)
		}
	}()
	fn()
	return
}

// collectHooks 泛型辅助：在锁内收集 key 对应的钩子函数快照，解锁后返回。
func collectHooks[T any](lock *sync.RWMutex, hooks map[string]func(T), keys []string) []func(T) {
	lock.RLock()
	fns := make([]func(T), 0, len(keys))
	for _, key := range keys {
		if fn, ok := hooks[key]; ok {
			fns = append(fns, fn)
		}
	}
	lock.RUnlock()
	return fns
}

// callPromiseHooks 通过钩子类型调用 Promise 事件钩子。
//
// 钩子函数在锁外执行，且以 safeCall(HookPanic) 包裹：钩子自身 panic 时
// 触发 HookPanic 事件，不影响调用方。
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

	for _, fn := range collectHooks(&hk.promiseHooksLock, hk.promiseHooks, keys) {
		hk.safeCall(HookPanic, func() { fn(p) })
	}
}

// callPanicHooks 通过钩子类型调用 Panic 事件钩子，接收 panic 值。
//
// 钩子函数在锁外执行。普通 panic 钩子以 safeCall(HookPanic) 包裹（钩子 panic
// 时触发 HookPanic）；而 HookPanic 钩子以 safeCall(ignorePanic) 包裹，
// 钩子 panic 时静默吞掉，避免递归触发造成啸叫。
func (hk *hooks) callPanicHooks(event HookType, r any) {
	var keys []string
	switch event {
	case AllPanic:
		keys = hk.allPanicHookKeys
	case PromisePanic:
		keys = hk.promisePanicHookKeys
	case AsyncPanic:
		keys = hk.asyncPanicHookKeys
	case HookPanic:
		keys = hk.hookPanicHookKeys
	case TimerPanic:
		keys = hk.timerPanicHookKeys
	}

	wrap := HookPanic
	if event == HookPanic {
		wrap = ignorePanic
	}
	for _, fn := range collectHooks(&hk.panicHooksLock, hk.panicHooks, keys) {
		hk.safeCall(wrap, func() { fn(r) })
	}
}
