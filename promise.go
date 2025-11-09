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

// ====== 实现类 ======
type promiseImpl struct {
	ip.Promise
	state           string
	value           any
	settledHandlers []handler
	done            chan struct{}
	manager         *PromiseManager
}

// 实现 Promise 接口
func (prom *promiseImpl) State() string {
	return prom.state
}

func (prom *promiseImpl) Result() any {
	return prom.value
}

func (prom *promiseImpl) Done() chan struct{} {
	return prom.done
}

// 实现 Promise 接口
func (prom *promiseImpl) Then(onFulfilled ip.ThenCallback, onRejected ip.ThenCallback) ip.Promise {
	prom2 := prom.manager.New(ip.Executor(func(resolve, reject func(v any)) error {
		return nil
	}))
	prom.settledHandlers = append(prom.settledHandlers, handler{
		onFulfilled: onFulfilled,
		onRejected:  onRejected,
		prom:        prom2.(*promiseImpl),
	})
	prom.manager.addTask(func() {
		prom.manager.flushHandlers(prom)
	})
	// 2.2.7 then 方法返回的新 Promise 实例的状态由回调函数的执行结果决定
	return prom2
}

// ref https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/catch
func (prom *promiseImpl) Catch(onRejected ip.ThenCallback) ip.Promise {
	return prom.Then(nil, onRejected)
}

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

// 实现 Stringer 接口
func (prom *promiseImpl) String() string {
	return fmt.Sprintf("Promise<%s>, result: %v", prom.state, prom.value)
}
