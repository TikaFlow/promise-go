package promise

import (
	"math/rand"
	"sync"
)

type HookType string

const (
	/*
		PromiseCreated 当 Promise 实例被创建时调用。
	*/
	PromiseCreated HookType = "created"

	/*
		PromiseChained 当 Promise 实例被链式调用时调用。
	*/
	PromiseChained HookType = "chained"

	/*
		PromiseFulfilled 当 Promise 实例被成功解决时调用。
	*/
	PromiseFulfilled HookType = "fulfilled"

	/*
		PromiseRejected 当 Promise 实例被拒绝时调用。
	*/
	PromiseRejected HookType = "rejected"

	/*
		PromiseSettled 当 Promise 实例被解决（无论成功或拒绝）时调用。
	*/
	PromiseSettled HookType = "settled"
)

type promiseHooks struct {
	createdHookKeys   []string
	chainedHookKeys   []string
	fulfilledHookKeys []string
	rejectedHookKeys  []string
	settledHookKeys   []string
	hooks             map[string]func(p Promise)
	hooksLock         sync.RWMutex
}

func (hk *promiseHooks) callHook(slice []string, p Promise) {
	for _, key := range slice {
		hk.hooks[key](p)
	}
}

func (hk *promiseHooks) callHooks(event HookType, p Promise) {
	hk.hooksLock.RLock()
	defer hk.hooksLock.RUnlock()

	switch event {
	case PromiseCreated:
		hk.callHook(hk.createdHookKeys, p)
	case PromiseChained:
		hk.callHook(hk.chainedHookKeys, p)
	case PromiseFulfilled:
		hk.callHook(hk.fulfilledHookKeys, p)
	case PromiseRejected:
		hk.callHook(hk.rejectedHookKeys, p)
	case PromiseSettled:
		hk.callHook(hk.settledHookKeys, p)
	}
}

/*
On 注册一个 Promise 钩子函数。
  - event: 钩子事件类型，可选值为 [PromiseCreated]/[PromiseChained]/[PromiseFulfilled]/[PromiseRejected]/[PromiseSettled]。
  - hook: 钩子函数，当事件触发时调用，并传入触发的 Promise 实例作为参数。

返回值：注册成功返回钩子函数的唯一标识，可用于后续移除钩子函数，失败返回空字符串。
*/
func (hk *promiseHooks) On(event HookType, hook func(p Promise)) string {
	if hook == nil {
		return ""
	}

	hk.hooksLock.Lock()
	defer hk.hooksLock.Unlock()

	var exist = true
	var key string
	for exist {
		key = string(event) + randString(16)
		_, exist = hk.hooks[key]
	}

	switch event {
	case PromiseCreated:
		hk.createdHookKeys = append(hk.createdHookKeys, key)
		hk.hooks[key] = hook
	case PromiseChained:
		hk.chainedHookKeys = append(hk.chainedHookKeys, key)
		hk.hooks[key] = hook
	case PromiseFulfilled:
		hk.fulfilledHookKeys = append(hk.fulfilledHookKeys, key)
		hk.hooks[key] = hook
	case PromiseRejected:
		hk.rejectedHookKeys = append(hk.rejectedHookKeys, key)
		hk.hooks[key] = hook
	case PromiseSettled:
		hk.settledHookKeys = append(hk.settledHookKeys, key)
		hk.hooks[key] = hook
	}

	return key
}

/*
Off 移除一个 Promise 钩子函数。
  - event: 钩子事件类型，可选值为 [PromiseCreated]/[PromiseChained]/[PromiseFulfilled]/[PromiseRejected]/[PromiseSettled]。
  - key: 要移除的钩子函数的唯一标识，由 On 方法返回。
*/
func (hk *promiseHooks) Off(event HookType, key string) {
	if key == "" {
		return
	}

	hk.hooksLock.Lock()
	defer hk.hooksLock.Unlock()

	var targetSlice *[]string

	switch event {
	case PromiseCreated:
		targetSlice = &hk.createdHookKeys
	case PromiseChained:
		targetSlice = &hk.chainedHookKeys
	case PromiseFulfilled:
		targetSlice = &hk.fulfilledHookKeys
	case PromiseRejected:
		targetSlice = &hk.rejectedHookKeys
	case PromiseSettled:
		targetSlice = &hk.settledHookKeys
	}

	if targetSlice != nil && deleteFromSlice(targetSlice, key) {
		delete(hk.hooks, key)
	}
}

func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return "@" + string(result)
}

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
