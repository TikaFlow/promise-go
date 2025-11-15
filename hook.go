package promise

import (
	"math/rand"
	"sync"
)

type PromiseHookType string

const (
	/*
		PromiseCreated 当 Promise 实例被创建时调用。
	*/
	PromiseCreated PromiseHookType = "created"

	/*
		PromiseChained 当 Promise 实例被链式调用时调用。
	*/
	PromiseChained PromiseHookType = "chained"

	/*
		PromiseFulfilled 当 Promise 实例被成功解决时调用。
	*/
	PromiseFulfilled PromiseHookType = "fulfilled"

	/*
		PromiseRejected 当 Promise 实例被拒绝时调用。
	*/
	PromiseRejected PromiseHookType = "rejected"

	/*
		PromiseSettled 当 Promise 实例被解决（无论成功或拒绝）时调用。
	*/
	PromiseSettled PromiseHookType = "settled"
)

var (
	createdHoookKeys   []string = make([]string, 0, 64)
	chainedHoookKeys   []string = make([]string, 0, 64)
	fulfilledHoookKeys []string = make([]string, 0, 64)
	rejectedHoookKeys  []string = make([]string, 0, 64)
	settledHoookKeys   []string = make([]string, 0, 64)

	hooks map[string]func(p Promise) = make(map[string]func(p Promise), 512)

	hooksLock sync.RWMutex
)

func callHook(slice []string, p Promise) {
	for _, key := range slice {
		hooks[key](p)
	}
}

func callHooks(event PromiseHookType, p Promise) {
	hooksLock.RLock()
	defer hooksLock.RUnlock()

	switch event {
	case PromiseCreated:
		callHook(createdHoookKeys, p)
	case PromiseChained:
		callHook(chainedHoookKeys, p)
	case PromiseFulfilled:
		callHook(fulfilledHoookKeys, p)
	case PromiseRejected:
		callHook(rejectedHoookKeys, p)
	case PromiseSettled:
		callHook(settledHoookKeys, p)
	}
}

/*
On 注册一个 Promise 钩子函数。
  - event: 钩子事件类型，可选值为 [PromiseCreated]/[PromiseChained]/[PromiseFulfilled]/[PromiseRejected]/[PromiseSettled]。
  - hook: 钩子函数，当事件触发时调用，并传入触发的 Promise 实例作为参数。

返回值：注册成功返回钩子函数的唯一标识，可用于后续移除钩子函数，失败返回空字符串。
*/
func On(event PromiseHookType, hook func(p Promise)) string {
	if hook == nil {
		return ""
	}

	hooksLock.Lock()
	defer hooksLock.Unlock()

	var exist = true
	var key string
	for exist {
		key = string(event) + randString(13)
		_, exist = hooks[key]
	}

	switch event {
	case PromiseCreated:
		createdHoookKeys = append(createdHoookKeys, key)
		hooks[key] = hook
	case PromiseChained:
		chainedHoookKeys = append(chainedHoookKeys, key)
		hooks[key] = hook
	case PromiseFulfilled:
		fulfilledHoookKeys = append(fulfilledHoookKeys, key)
		hooks[key] = hook
	case PromiseRejected:
		rejectedHoookKeys = append(rejectedHoookKeys, key)
		hooks[key] = hook
	case PromiseSettled:
		settledHoookKeys = append(settledHoookKeys, key)
		hooks[key] = hook
	}

	return key
}

/*
Off 移除一个 Promise 钩子函数。
  - event: 钩子事件类型，可选值为 [PromiseCreated]/[PromiseChained]/[PromiseFulfilled]/[PromiseRejected]/[PromiseSettled]。
  - key: 要移除的钩子函数的唯一标识，由 On 方法返回。
*/
func Off(event PromiseHookType, key string) {
	if key == "" {
		return
	}

	hooksLock.Lock()
	defer hooksLock.Unlock()

	var targetSlice *[]string

	switch event {
	case PromiseCreated:
		targetSlice = &createdHoookKeys
	case PromiseChained:
		targetSlice = &chainedHoookKeys
	case PromiseFulfilled:
		targetSlice = &fulfilledHoookKeys
	case PromiseRejected:
		targetSlice = &rejectedHoookKeys
	case PromiseSettled:
		targetSlice = &settledHoookKeys
	}

	if targetSlice != nil && deleteFromSlice(targetSlice, key) {
		delete(hooks, key)
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
