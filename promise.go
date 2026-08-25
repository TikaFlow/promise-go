package promise

import (
	"fmt"
	"sync"
)

// Promise 一个拥有 then 方法的对象，其行为符合 Promises/A+ 规范
//
// # 通过 nil 调用 Promise 的任何方法都可能触发 panic
type Promise struct {
	value           any
	reason          error
	state           string
	dataLock        sync.RWMutex
	settledHandlers []*handler
	settled         chan struct{}
	resolved        sync.Once
	// 所属的事件循环
	eventLoop *EventLoop
}

// 待处理的 Promise 回调
type handler struct {
	onFulfilled ThenCallback
	onRejected  CatchCallback

	// 即将返回的 Promise 实例（新）
	prom *Promise
}

// addHandler 将回调处理器追加到待处理列表。注册可无限次、永不阻塞（修正固定缓冲上限导致的死锁）。
func (prom *Promise) addHandler(h *handler) {
	prom.dataLock.Lock()
	prom.settledHandlers = append(prom.settledHandlers, h)
	prom.dataLock.Unlock()
}

// takeHandlers 原子地取出并清空全部待处理回调，按注册顺序返回。
func (prom *Promise) takeHandlers() []*handler {
	prom.dataLock.Lock()
	hs := prom.settledHandlers
	prom.settledHandlers = nil
	prom.dataLock.Unlock()
	return hs
}

// State 返回 [Promise] 的当前状态
func (prom *Promise) State() string {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.state
}

// Done 返回一个通道，当 [Promise] 状态变为 [Fulfilled] 或 [Rejected] 时，该通道会被关闭
//
// 不建议无限等待，因为规范允许 [Promise] 永远保持 [Pending] 状态
func (prom *Promise) Done() <-chan struct{} {
	return prom.settled
}

// Value 返回 [Promise] 的结果值，如果 [Promise] 当前状态不是 [Fulfilled]，则值为 nil
func (prom *Promise) Value() any {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.value
}

// Reason 返回 [Promise] 的拒绝理由，如果 [Promise] 当前状态不是 [Rejected]，则值为 nil
func (prom *Promise) Reason() error {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.reason
}

// Then 返回一个新的 [Promise]，其状态和结果值由 onFulfilled 或 onRejected 回调函数的执行结果决定
func (prom *Promise) Then(onFulfilled ThenCallback, onRejected CatchCallback) *Promise {
	prom2 := prom.eventLoop.NewPromise(func(resolve, reject func(v any)) error {
		return nil
	})
	prom.eventLoop.hooks.callPromiseHooks(PromiseChained, prom)
	prom.addHandler(&handler{
		onFulfilled: onFulfilled,
		onRejected:  onRejected,
		prom:        prom2,
	})

	if prom.State() != Pending {
		flushHandlers(prom)
	}
	// 2.2.7 then 方法返回的新 Promise 实例的状态由回调函数的执行结果决定
	return prom2
}

// Catch 返回一个新的 [Promise]，其状态和结果值由 onRejected 回调函数的执行结果决定，详见 [MDN]
//
// 这是一个语法糖，等价于以下语句：
//
//	promise.Then(nil, onRejected)
//
// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/catch
func (prom *Promise) Catch(onRejected CatchCallback) *Promise {
	return prom.Then(nil, onRejected)
}

// Finally 返回一个新的 [Promise]，其状态和结果与原 [Promise] 相同，以下情况除外：
//   - onFinally 抛出异常 e，则以 e 为理由拒绝新 [Promise]
//   - onFinally 返回一个已拒绝的 [Promise] 实例，则以同样的理由拒绝新 [Promise]
//
// onFinally 返回的未决/已解决 [Promise]：其返回值被丢弃，新 [Promise] 仍沿用原已决状态
// （不等待该 [Promise]；与 MDN 的 Promise.resolve(onFinally()).then(...) 在“等待未决”上不同）。
func (prom *Promise) Finally(onFinally FinallyCallback) *Promise {
	pass := func(val any, reason error) (any, error) {
		if onFinally == nil {
			return val, reason
		}

		res, err := onFinally()
		if err != nil {
			return nil, err
		}

		if p, ok := res.(*Promise); ok && p.State() == Rejected {
			return nil, p.Reason()
		}

		return val, reason
	}
	return prom.Then(
		func(v any) (any, error) { return pass(v, nil) },
		func(r error) (any, error) { return pass(nil, r) },
	)
}

// String [fmt.Stringer.String] 接口实现
func (prom *Promise) String() string {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()

	if prom.state == Pending {
		return fmt.Sprintf("Promise<%s>", prom.state)
	}
	if prom.state == Rejected {
		return fmt.Sprintf("Promise<%s>, reason: %v", prom.state, prom.reason)
	}
	return fmt.Sprintf("Promise<%s>, value: %v", prom.state, prom.value)
}
