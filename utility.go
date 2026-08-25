package promise

// Async 将 fn 作为一个异步任务执行
//
// 类似于 `go fn()`，但会在一个专用的 worker-pool 中进行，且能获取返回值
//
// # return
//
// 一个新的 [Promise] 实例，并在 fn 函数执行完成后变为解决状态，解决值是 fn 的返回值
// 若 fn 函数抛出异常 err，则 [Promise] 实例会被拒绝，且拒绝理由为 err
func (el *EventLoop) Async(fn func() (any, error)) *Promise {
	if fn == nil {
		return el.Reject(NewTypeError("fn must be a function"))
	}

	return el.NewPromise(func(resolve, reject func(any)) error {
		task := func() {
			// fn 内 panic 视为拒绝理由，且触发 AsyncPanic 钩子
			var v any
			var err error
			if r := el.hooks.safeCall(AsyncPanic, func() {
				v, err = fn()
			}); r != nil {
				reject(r)
				return
			}
			if err != nil {
				reject(err)
				return
			}
			resolve(v)
		}
		el.pushTask(task)
		return nil
	})
}

// Await 等待 Promise 已决并获取解决值，可设定超时时间，以免无限等待
//   - prom 需要等待的 [Promise] 实例，如果不是 [Promise] 实例，则会被直接返回
//   - timeout 超时时间，单位为毫秒
//
// # return
//
//   - v 目标 prom 的解决值
//   - err 拒绝理由，当 err 存在时，代表 [Promise] 被拒绝，此时 v 的值无意义
//
// 警告：Await 会阻塞当前 goroutine。不建议在事件循环回调（如 Then/定时器回调）中调用，
// 否则会阻塞事件循环，甚至死锁。
func (el *EventLoop) Await(prom any, timeout int64) (v any, err error) {
	if timeout <= 0 {
		return nil, NewRangeError("await timeout must be greater than 0")
	}

	prom2, ok := prom.(*Promise)
	if !ok {
		return prom, nil
	}

	var timerID int
	wait := el.NewPromise(func(resolve, reject func(v any)) error {
		timerID = el.SetTimeout(func() {
			reject(NewTimeoutError("await timeout"))
		}, timeout)
		return nil
	})

	select {
	case <-prom2.Done():
		el.ClearTimeout(timerID)
		if prom2.State() == Rejected {
			err = prom2.Reason()
		} else {
			v = prom2.Value()
		}
	case <-wait.Done():
		if wait.State() == Rejected {
			err = wait.Reason()
		} else {
			v = wait.Value()
		}
	}

	return
}

// NewPromise 创建一个新的 [Promise] 实例
//   - exec 执行器函数，用于定义 [Promise] 的异步操作
func (el *EventLoop) NewPromise(exec Executor) *Promise {
	if exec == nil {
		panic(NewTypeError("Promise executor must be a function"))
	}

	prom := &Promise{
		value:           nil,
		reason:          nil,
		state:           Pending,
		settledHandlers: make([]*handler, 0),
		settled:         make(chan struct{}),
		eventLoop:       el,
	}
	el.hooks.callPromiseHooks(PromiseCreated, prom)

	res := func(data any) {
		prom.resolved.Do(func() {
			resolvePromise(prom, data)
		})
	}
	rej := func(reason any) {
		prom.resolved.Do(func() {
			rejectPromise(prom, reason)
		})
	}

	// 执行器内部发生 panic 时，同样按照规范将其作为拒绝理由处理，
	// 并触发 ExecutorPanic 钩子。注：若执行器先 resolve 再 panic，
	// sync.Once 保证该 panic 被忽略（规范 2.3.3 已决后忽略）。
	if r := el.hooks.safeCall(ExecutorPanic, func() {
		if err := exec(res, rej); err != nil {
			rej(err)
		}
	}); r != nil {
		rej(r)
	}
	return prom
}

// PromiseWithResolvers 创建一个新的 [Promise] 实例，同时返回 resolve 和 reject 函数，
// 对应于传入给 Promise() 构造函数执行器的两个参数
//
// 这使得可以在 [Promise] 外部手动解决或拒绝 [Promise]，详见 [MDN]
//
// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/withResolvers#%E6%8F%8F%E8%BF%B0
func (el *EventLoop) PromiseWithResolvers() (*Promise, func(any), func(any)) {
	var resolve, reject func(any)
	p := el.NewPromise(func(res, rej func(v any)) error {
		resolve = res
		reject = rej
		return nil
	})
	return p, resolve, reject
}

// QueueMicrotask 将回调函数添加到微任务队列末尾
func (el *EventLoop) QueueMicrotask(fn func()) {
	if fn == nil {
		return
	}
	el.microtaskQueue.Push(fn)
}

// Reject 返回一个已拒绝的 [Promise]，拒绝理由为指定值 reason，详见 [MDN]
//
// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/reject
func (el *EventLoop) Reject(reason any) *Promise {
	return el.NewPromise(func(resolve, reject func(v any)) error {
		reject(reason)
		return nil
	})
}

// Resolve 返回一个已解决的 [Promise]，解决值为指定值 value
//
// 如果 value 已经是 [Promise]，则直接返回该 [Promise]，详见 [MDN]
//
// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/resolve
func (el *EventLoop) Resolve(value any) *Promise {
	if prom, ok := value.(*Promise); ok {
		return prom
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		resolve(value)
		return nil
	})
}

// Try 接受一个任意类型的回调函数，并将其结果封装成一个 [Promise]，详见 [MDN]。
//   - fn 任意类型的回调函数，接受任意数量的参数，函数返回值格式为 (any, error)
//   - args 将要传递给 fn 函数的参数列表
//
// # return
//
// 一个新的 [Promise]，其状态将会是：
//   - 已解决（[Fulfilled]）：如果 fn 函数返回一个普通值
//   - 已拒绝（[Rejected]）：如果 fn 函数返回了 err
//   - 异步解决或拒绝：如果 fn 函数返回一个 [Promise]，新 [Promise] 会吸收该 [Promise] 的状态
//
// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/try
func (el *EventLoop) Try(fn func(...any) (any, error), args ...any) *Promise {
	return el.NewPromise(func(resolve, reject func(v any)) error {
		if fn == nil {
			return NewTypeError("Promise executor must be a function")
		}

		result, err := fn(args...)
		if err != nil {
			reject(err)
			return nil
		}
		resolve(result)
		return nil
	})
}
