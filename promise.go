package promise

import (
	"errors"
	"fmt"
	"sync"
)

/*
handler 表示待处理的 Promise 回调。
*/
type handler struct {
	onFulfilled ThenCallback
	onRejected  ThenCallback

	/*
	   即将返回的 Promise 实例（新）
	*/
	prom *promiseImpl
}

/*
promiseImpl 表示 Promise 的具体实现类。
*/
type promiseImpl struct {
	state           string
	result          any
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
[Promise.Result]
*/
func (prom *promiseImpl) Result() any {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.result
}

/*
[Promise.Done]
*/
func (prom *promiseImpl) Done() <-chan struct{} {
	return prom.settled
}

/*
[Promise.Then]
*/
func (prom *promiseImpl) Then(onFulfilled ThenCallback, onRejected ThenCallback) Promise {
	prom2 := New(func(resolve, reject func(v any)) error {
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
func (prom *promiseImpl) Catch(onRejected ThenCallback) Promise {
	return prom.Then(nil, onRejected)
}

/*
[Promise.Finally]
*/
func (prom *promiseImpl) Finally(onFinally FinallyCallback) Promise {
	cb := func(v any) (any, error) {
		if onFinally == nil {
			return v, nil
		}

		res, err := onFinally()
		if err != nil {
			return res, err
		}

		if result, ok := res.(Promise); ok {
			if result.State() == Rejected {
				reason := result.Result()
				return reason, errors.New("finally callback returns a rejected Promise")
			}
		}

		return v, nil
	}
	return prom.Then(cb, cb)
}

/*
[fmt.Stringer.String]
*/
func (prom *promiseImpl) String() string {
	return fmt.Sprintf("Promise<%s>, result: %v", prom.State(), prom.Result())
}
