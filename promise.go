package promise

import (
	"errors"
	"fmt"
	ip "github.com/TikaFlow/promise-go/ipromise"
)

// handler 表示待处理的 Promise 回调
type handler struct {
	onFulfilled ip.ThenCallback
	onRejected  ip.ThenCallback
	prom        *promiseImpl // 即将返回的 Promise 实例（新）
}

// promiseImpl 表示 Promise 的具体实现类
type promiseImpl struct {
	ip.Promise
	state           string
	value           any
	settledHandlers chan *handler
	done            chan struct{}
}

// State 返回 Promise 的当前状态。
func (prom *promiseImpl) State() string {
	return prom.state
}

// Result 返回 Promise 的结果值。
func (prom *promiseImpl) Result() any {
	return prom.value
}

// Done 返回一个通道，当 Promise 状态变为 Fulfilled 或 Rejected 时，该通道会被关闭。
func (prom *promiseImpl) Done() chan struct{} {
	return prom.done
}

// Then 方法返回一个新的 Promise，其状态和结果值由 onFulfilled 或 onRejected 回调函数的执行结果决定。
func (prom *promiseImpl) Then(onFulfilled ip.ThenCallback, onRejected ip.ThenCallback) ip.Promise {
	prom2 := New(func(resolve, reject func(v any)) error {
		return nil
	})
	prom.settledHandlers <- &handler{
		onFulfilled: onFulfilled,
		onRejected:  onRejected,
		prom:        prom2.(*promiseImpl),
	}

	if prom.state != ip.Pending {
		QueueMicrotask(func() {
			flushHandlers(prom)
		})
	}
	// 2.2.7 then 方法返回的新 Promise 实例的状态由回调函数的执行结果决定
	return prom2
}

// Catch 方法返回一个新的 Promise，其状态和结果值由 onRejected 回调函数的执行结果决定。
// ref https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/catch
func (prom *promiseImpl) Catch(onRejected ip.ThenCallback) ip.Promise {
	return prom.Then(nil, onRejected)
}

// Finally 方法返回一个新的 Promise，其状态和结果值与原 Promise 相同，以下情况除外：
// - onFinally 抛出异常e，则以 e 为理由拒绝新 Promise;
// - onFinally 返回一个拒绝的 Promise 实例，则以同样的理由拒绝新 Promise。
func (prom *promiseImpl) Finally(onFinally ip.FinallyCallback) ip.Promise {
	cb := func(v any) (any, error) {
		// 默认穿透
		if onFinally == nil {
			return v, nil
		}

		res, err := onFinally()
		// 报错则拒绝
		if err != nil {
			return res, err
		}

		// 是一个拒绝的 Promise 实例，则以同样的理由拒绝
		if result, ok := res.(ip.Promise); ok {
			if result.State() == ip.Rejected {
				reason := result.Result()
				return reason, errors.New("finally callback returns a rejected Promise")
			}
		}

		// 其他情况忽略 onFinally 的返回值
		return v, nil
	}
	return prom.Then(cb, cb)
}

// String 返回 Promise 的字符串表示，包含状态和结果值。
func (prom *promiseImpl) String() string {
	return fmt.Sprintf("Promise<%s>, result: %v", prom.state, prom.value)
}
