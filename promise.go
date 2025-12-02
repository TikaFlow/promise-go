package promise

import (
	"fmt"
	"sync"
)

/*
handler 表示待处理的 Promise 回调。
*/
type handler struct {
	onFulfilled ThenCallback
	onRejected  CatchCallback

	/*
	   即将返回的 Promise 实例（新）
	*/
	prom *promiseImpl
}

/*
promiseImpl 表示 Promise 的具体实现类。
*/
type promiseImpl struct {
	value           any
	reason          error
	state           string
	dataLock        sync.RWMutex
	settledHandlers chan *handler
	settled         chan struct{}
	resolved        sync.Once
}

/*
[Promise.State]
*/
func (prom *promiseImpl) State() string {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.state
}

/*
[Promise.Done]
*/
func (prom *promiseImpl) Done() <-chan struct{} {
	return prom.settled
}

/*
[Promise.Value]
*/
func (prom *promiseImpl) Value() any {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.value
}

/*
[Promise.Reason]
*/
func (prom *promiseImpl) Reason() error {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.reason
}

/*
[Promise.Then]
*/
func (prom *promiseImpl) Then(onFulfilled ThenCallback, onRejected CatchCallback) Promise {
	prom2 := New(func(resolve func(v any), reject func(r error)) error {
		return nil
	})
	callHooks(PromiseChained, prom)
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

/*
[Promise.Catch]
*/
func (prom *promiseImpl) Catch(onRejected CatchCallback) Promise {
	return prom.Then(nil, onRejected)
}

/*
[Promise.Finally]
*/
func (prom *promiseImpl) Finally(onFinally FinallyCallback) Promise {
	res_cb := func(v any) (any, error) {
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
	rej_cb := func(r error) (any, error) {
		return res_cb(r)
	}

	return prom.Then(res_cb, rej_cb)
}

/*
[fmt.Stringer.String]
*/
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
