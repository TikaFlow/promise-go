package promise

// 解决一个 [Promise]
func resolvePromise(prom *Promise, value any) {
	if prom == nil {
		return
	}

	if prom.State() != Pending {
		return
	}

	if value == prom {
		// 2.3.1 如果 Promise 和已决值相同，则抛出 TypeError 异常
		rejectPromise(prom, NewTypeError("Chaining cycle detected for promise"))
		return
	}

	// 2.3.2
	if x, ok := value.(*Promise); ok {
		// 2.3.2 如果已决值是 Promise 对象，则采用其状态
		prom.eventLoop.QueueMicrotask(func() {
			x.Then(func(v any) (any, error) {
				resolvePromise(prom, v)
				return nil, nil
			}, func(r error) (any, error) {
				rejectPromise(prom, r)
				return nil, nil
			})
		})
		return
	}
	// 2.3.3 同上

	// 2.3.4 其他情况，则使用 value 作为已决值
	// 状态判定 + close 放入同一把写锁临界区，结构上避免"双 close"。
	prom.dataLock.Lock()
	if prom.state != Pending {
		prom.dataLock.Unlock()
		return
	}
	prom.state = Fulfilled
	prom.value = value
	close(prom.settled)
	prom.dataLock.Unlock()
	prom.eventLoop.hooks.callPromiseHooks(PromiseSettled, prom)
	prom.eventLoop.hooks.callPromiseHooks(PromiseFulfilled, prom)
	flushHandlers(prom)
}

// 拒绝一个 [Promise]
func rejectPromise(prom *Promise, r any) {
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

	// 状态判定 + close 放入同一把写锁临界区，结构上避免"双 close"。
	prom.dataLock.Lock()
	if prom.state != Pending {
		prom.dataLock.Unlock()
		return
	}
	prom.state = Rejected
	prom.reason = reason
	close(prom.settled)
	prom.dataLock.Unlock()
	prom.eventLoop.hooks.callPromiseHooks(PromiseSettled, prom)
	prom.eventLoop.hooks.callPromiseHooks(PromiseRejected, prom)
	flushHandlers(prom)
}

// 调用 [Promise] 的回调函数
func flushHandlers(cur *Promise) {
	// 2.2.6 then 可以注册多次，且会按照注册顺序执行
	for _, hdl := range cur.takeHandlers() {
		job := func() {
			// 2.2.7.2 回调通过 panic 抛出的异常同样视为拒绝理由（与返回 err 等价）
			defer func() {
				if r := recover(); r != nil {
					rejectPromise(hdl.prom, r)
				}
			}()

			var res any
			var err error
			if cur.State() == Fulfilled {
				if hdl.onFulfilled == nil {
					// 2.2.1 如果回调不是函数，则忽略（穿透->2.2.7.3）
					resolvePromise(hdl.prom, cur.Value())
					return
				}

				// 2.2.2
				res, err = hdl.onFulfilled(cur.Value())
				if err != nil {
					// 2.2.7.2 如果回调函数抛出一个异常 e，则新 Promise 实例必须被拒绝，且拒绝原因为 e
					rejectPromise(hdl.prom, err)
					return
				}
			} else { // 必然是 Rejected
				if hdl.onRejected == nil {
					// 2.2.1 如果回调不是函数，则忽略（穿透->2.2.7.4）
					rejectPromise(hdl.prom, cur.Reason())
					return
				}

				// 2.2.3
				res, err = hdl.onRejected(cur.Reason())
				if err != nil {
					// 2.2.7.2 如果回调函数抛出一个异常 e，则新 Promise 实例必须被拒绝，且拒绝原因为 e
					rejectPromise(hdl.prom, err)
					return
				}
			}

			// 2.2.7.1 如果回调函数返回一个值，则使用该值来 resolve 新 Promise
			resolvePromise(hdl.prom, res)
		}
		cur.eventLoop.QueueMicrotask(job)
	}
}
