package promise

import (
	"sync"
)

// HookType 钩子类型
type HookType string

// OnCreated 当 Promise 实例被创建时调用。
const OnCreated HookType = "created"

// OnChained 当 Promise 实例被链式调用时调用。
const OnChained HookType = "chained"

// OnFulfilled 当 Promise 实例解决时调用。
const OnFulfilled HookType = "fulfilled"

// OnRejected 当 Promise 实例拒绝时调用。
const OnRejected HookType = "rejected"

// OnSettled 当 Promise 实例被解决（无论成功或拒绝）时调用。
const OnSettled HookType = "settled"

// 钩子实例定义
type promiseHooks struct {
	createdHookKeys   []string
	chainedHookKeys   []string
	fulfilledHookKeys []string
	rejectedHookKeys  []string
	settledHookKeys   []string
	hooks             map[string]func(p *Promise)
	hooksLock         sync.RWMutex
}

// 通过钩子类型调用钩子。
//
// 钩子函数在锁外执行：先在锁内按 key 收集函数快照，再释放读锁、逐个执行。
// 因此钩子内可安全调用 [EventLoop.On] / [EventLoop.Off]（二者需要写锁），
// 不会因"持有读锁时尝试升级为写锁"而自死锁。
func (hk *promiseHooks) callHooks(event HookType, p *Promise) {
	hk.hooksLock.RLock()
	// 锁内只读取：从 map 收集钩子引用。Off 在快照之前发生则本钩子不会触发；
	// 在快照之后发生则本钩子仍会执行一次（可接受的竞态）。
	var keys []string
	switch event {
	case OnCreated:
		keys = hk.createdHookKeys
	case OnChained:
		keys = hk.chainedHookKeys
	case OnFulfilled:
		keys = hk.fulfilledHookKeys
	case OnRejected:
		keys = hk.rejectedHookKeys
	case OnSettled:
		keys = hk.settledHookKeys
	}

	fns := make([]func(p *Promise), 0, len(keys))
	for _, key := range keys {
		if fn, ok := hk.hooks[key]; ok {
			fns = append(fns, fn)
		}
	}
	hk.hooksLock.RUnlock()

	for _, fn := range fns {
		fn(p)
	}
}
