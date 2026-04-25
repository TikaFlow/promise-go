package promise

import (
	"math/rand"
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

// On [EventLoop.On]
func (el *eventLoopImpl) On(event HookType, hook func(p Promise)) string {
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

// Off [EventLoop.Off]
func (el *eventLoopImpl) Off(event HookType, key string) bool {
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

// 获取一个指定长度的随机字符串
func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return "@" + string(result)
}

// 从 slice 中删除内容为 key 的元素
func deleteFromSlice(slice *[]string, key string) bool {
	target := false
	for i, k := range *slice {
		if k == key {
			target = true
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			break
		}
	}
	return target
}
