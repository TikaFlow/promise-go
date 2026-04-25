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

// State [Promise.State]
func (prom *promiseImpl) State() string {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.state
}

// Done [Promise.Done]
func (prom *promiseImpl) Done() <-chan struct{} {
	return prom.settled
}

// Value [Promise.Value]
func (prom *promiseImpl) Value() any {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.value
}

// Reason [Promise.Reason]
func (prom *promiseImpl) Reason() error {
	prom.dataLock.RLock()
	defer prom.dataLock.RUnlock()
	return prom.reason
}

// Then [Promise.Then]
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

// Catch [Promise.Catch]
func (prom *promiseImpl) Catch(onRejected CatchCallback) Promise {
	return prom.Then(nil, onRejected)
}

// Finally [Promise.Finally]
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

// String [fmt.Stringer.String]
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
