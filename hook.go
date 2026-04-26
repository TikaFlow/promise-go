package promise

import (
	"sync"
)

// HookType 钩子类型
type HookType string

const (
	// OnCreated 当 Promise 实例被创建时调用。
	OnCreated HookType = "created"

	// OnChained 当 Promise 实例被链式调用时调用。
	OnChained HookType = "chained"

	// OnFulfilled 当 Promise 实例解决时调用。
	OnFulfilled HookType = "fulfilled"

	// OnRejected 当 Promise 实例拒绝时调用。
	OnRejected HookType = "rejected"

	// OnSettled 当 Promise 实例被解决（无论成功或拒绝）时调用。
	OnSettled HookType = "settled"
)

// 钩子实例定义
type promiseHooks struct {
	createdHookKeys   []string
	chainedHookKeys   []string
	fulfilledHookKeys []string
	rejectedHookKeys  []string
	settledHookKeys   []string
	hooks             map[string]func(p Promise)
	hooksLock         sync.RWMutex
}

// 调用具体钩子
func (hk *promiseHooks) callHook(slice []string, p Promise) {
	for _, key := range slice {
		hk.hooks[key](p)
	}
}

// 通过钩子类型调用钩子
func (hk *promiseHooks) callHooks(event HookType, p Promise) {
	hk.hooksLock.RLock()
	defer hk.hooksLock.RUnlock()

	switch event {
	case OnCreated:
		hk.callHook(hk.createdHookKeys, p)
	case OnChained:
		hk.callHook(hk.chainedHookKeys, p)
	case OnFulfilled:
		hk.callHook(hk.fulfilledHookKeys, p)
	case OnRejected:
		hk.callHook(hk.rejectedHookKeys, p)
	case OnSettled:
		hk.callHook(hk.settledHookKeys, p)
	}
}
