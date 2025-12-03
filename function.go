package promise

import (
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

func resolvePromise(prom *promiseImpl, value any) {
	if getGoroutineID() != eventLoopID {
		SetTimeout(func() {
			resolvePromise(prom, value)
		}, 0)
		return
	}

	if prom == nil {
		return
	}

	if prom.State() != Pending {
		return
	}

	if value == prom {
		// 2.3.1 如果 Promise 和已决值相同，则抛出 TypeError 异常
		rejectPromsie(prom, NewTypeError("Chaining cycle detected for promise"))
		return
	}

	// 2.3.2
	if x, ok := value.(Promise); ok {
		// 2.3.2 如果已决值是 Promise 对象，则采用其状态
		QueueMicrotask(func() {
			x.Then(func(v any) (any, error) {
				resolvePromise(prom, v)
				return nil, nil
			}, func(r error) (any, error) {
				rejectPromsie(prom, r)
				return nil, nil
			})
		})
		return
	}
	// 2.3.3 同上

	// 2.3.4 其他情况，则使用 value 作为已决值
	prom.dataLock.Lock()
	prom.state = Fulfilled
	prom.value = value
	prom.dataLock.Unlock()
	close(prom.settled)
	callHooks(PromiseSettled, prom)
	callHooks(PromiseFulfilled, prom)
	flushHandlers(prom)
}

func rejectPromsie(prom *promiseImpl, r any) {
	if getGoroutineID() != eventLoopID {
		SetTimeout(func() {
			rejectPromsie(prom, r)
		}, 0)
		return
	}

	if prom == nil {
		return
	}

	if prom.State() != Pending {
		return
	}

	reason, ok := r.(error)
	if !ok {
		reason = NewUnexpectedError(r)
	}

	prom.dataLock.Lock()
	prom.state = Rejected
	prom.reason = reason
	prom.dataLock.Unlock()
	close(prom.settled)
	callHooks(PromiseSettled, prom)
	callHooks(PromiseRejected, prom)
	flushHandlers(prom)
}

func resetLoopTimer() {
	loopTimerLock.Lock()
	defer loopTimerLock.Unlock()

	if !loopTimer.Stop() {
		select {
		case <-loopTimer.C:
		default:
		}
	}
	timeoutLock.RLock()
	defer timeoutLock.RUnlock()
	loopTimer.Reset(timeout + margin)
}

func flushHandlers(cur *promiseImpl) {
	// 2.2.6 then 可以注册多次，且会按照注册顺序执行
	for {
		select {
		case hdl := <-cur.settledHandlers:
			job := func() {
				var res any
				var err error
				if cur.State() == Fulfilled {
					if hdl.onFulfilled == nil {
						// 2.2.1 如果回调不是函数，则忽略（穿透->2.2.7.3）
						resolvePromise(hdl.prom, cur.Value())
						return
					} else {
						// 2.2.2
						res, err = hdl.onFulfilled(cur.Value())
						if err != nil {
							// 2.2.7.2 如果回调函数抛出一个异常 e，则新 Promise 实例必须被拒绝，且拒绝原因为 e
							rejectPromsie(hdl.prom, err)
							return
						}
					}
				} else { // 必然是 Rejected
					if hdl.onRejected == nil {
						// 2.2.1 如果回调不是函数，则忽略（穿透->2.2.7.4）
						rejectPromsie(hdl.prom, cur.Reason())
						return
					} else {
						// 2.2.3
						res, err = hdl.onRejected(cur.Reason())
						if err != nil {
							// 2.2.7.2 如果回调函数抛出一个异常 e，则新 Promise 实例必须被拒绝，且拒绝原因为 e
							rejectPromsie(hdl.prom, err)
							return
						}
					}
				}

				// 2.2.7.1 如果回调函数返回一个值，则使用该值来 resolve 新 Promise
				resolvePromise(hdl.prom, res)
			}
			QueueMicrotask(job)
		default:
			return
		}
	}
}

func getGoroutineID() uint64 {
	buf := make([]byte, 64)
	n := runtime.Stack(buf, false)
	stackStr := string(buf[:n])

	firstLine := stackStr
	if idx := strings.Index(stackStr, "\n"); idx != -1 {
		firstLine = stackStr[:idx]
	}

	re := regexp.MustCompile(`goroutine (\d+) `)
	match := re.FindStringSubmatch(firstLine)
	if len(match) < 2 {
		return 0
	}

	id, _ := strconv.ParseUint(match[1], 10, 64)
	return id
}
