package promise

import (
	"fmt"
	"sync"
)

// 待处理的 Promise 回调
type handler struct {
	onFulfilled ThenCallback
	onRejected  CatchCallback

	// 即将返回的 Promise 实例（新）
	prom *promiseImpl
}

// 内部实现类
type promiseImpl struct {
	value           any
	reason          error
	state           string
	dataLock        sync.RWMutex
	settledHandlers chan *handler
	settled         chan struct{}
	resolved        sync.Once
	// 所属的事件循环
	eventLoop *eventLoopImpl
}

// State 返回 [Promise] 的当前状态
func (prom *promiseImpl) State() string {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.state
}

// Done 返回一个通道，当 [Promise] 状态变为 [Fulfilled] 或 [Rejected] 时，该通道会被关闭
//
// 不建议无限等待，因为规范允许 [Promise] 永远保持 [Pending] 状态
func (prom *promiseImpl) Done() <-chan struct{} {
	return prom.settled
}

// Value 返回 [Promise] 的结果值，如果 [Promise] 当前状态不是 [Fulfilled]，则值为 nil
func (prom *promiseImpl) Value() any {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.value
}

// Reason 返回 [Promise] 的拒绝理由，如果 [Promise] 当前状态不是 [Rejected]，则值为 nil
func (prom *promiseImpl) Reason() error {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.reason
}

// Then 返回一个新的 [Promise]，其状态和结果值由 onFulfilled 或 onRejected 回调函数的执行结果决定
func (prom *promiseImpl) Then(onFulfilled ThenCallback, onRejected CatchCallback) Promise {
	prom2 := prom.eventLoop.NewPromise(func(resolve, reject func(v any)) error {
		return nil
	})
	prom.eventLoop.hooks.callHooks(OnChained, prom)
	prom.settledHandlers <- &handler{
		onFulfilled: onFulfilled,
		onRejected:  onRejected,
		prom:        prom2.(*promiseImpl),
	}

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
func (prom *promiseImpl) Catch(onRejected CatchCallback) Promise {
	return prom.Then(nil, onRejected)
}

// Finally 返回一个新的 [Promise]，其状态和结果与原 [Promise] 相同，以下情况除外：
//   - onFinally 抛出异常 e，则以 e 为理由拒绝新 [Promise]
//   - onFinally 返回一个拒绝的 [Promise] 实例，则以同样的理由拒绝新 [Promise]
func (prom *promiseImpl) Finally(onFinally FinallyCallback) Promise {
	resCb := func(v any) (any, error) {
		if onFinally == nil {
			return v, nil
		}

		res, err := onFinally()
		if err != nil {
			return res, err
		}

		if result, ok := res.(Promise); ok {
			if result.State() == Rejected {
				reason := result.Reason()
				return nil, reason
			}
		}

		return v, nil
	}
	rejCb := func(r error) (any, error) {
		return resCb(r)
	}

	return prom.Then(resCb, rejCb)
}

// String [fmt.Stringer.String] 接口实现
func (prom *promiseImpl) String() string {
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

// DocPromise 仅用于生成 [Promise] 的文档，请勿使用此类型
//
// 文档中的 DocPromise 用于指代 [Promise]
type DocPromise struct {
	promiseImpl
}
