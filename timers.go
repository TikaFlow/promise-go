package promise

// ClearInterval 清除由 [EventLoop.SetInterval] 函数创建的定时器
//   - id 定时器ID
func (el *EventLoop) ClearInterval(id int) {
	if id == -1 {
		return
	}
	el.timeline.clearCh <- id
}

// ClearTimeout 清除由 [EventLoop.SetTimeout] 函数创建的定时器
//   - id 定时器ID
func (el *EventLoop) ClearTimeout(id int) {
	if id == -1 {
		return
	}
	el.timeline.clearCh <- id
}

// Delay 返回一个新的 [Promise]，其状态会在延迟时间后被解决
//   - prom 将会使用的解决值，如果 prom 是 [Promise] 实例，则会等待其已决后才开始计时；
//     如果是一个已拒绝的 [Promise]，则会立即拒绝新 [Promise]
//   - millis 延迟时间，单位为毫秒
func (el *EventLoop) Delay(prom any, millis int64) *Promise {
	return el.NewPromise(func(resolve, reject func(v any)) error {
		el.Resolve(prom).Then(func(v2 any) (any, error) {
			el.SetTimeout(func() {
				resolve(v2)
			}, millis)
			return nil, nil
		}, func(r error) (any, error) {
			reject(r)
			return nil, nil
		})
		return nil
	})
}

// Timeout 返回一个新的 [Promise]：
//   - 若 prom 在 millis 毫秒内未 settled，则以 *TimeoutError 拒绝新 [Promise]；
//   - 否则跟随 prom 的状态（同值 / 同理由）
//
// millis 负值按 0 处理（与 [EventLoop.SetTimeout] 一致）。它是框架层的显式超时组合子：
// 按需为某个 promise 配置超时，不影响其他 promise，也不改动 [Promise] 结构。
func (el *EventLoop) Timeout(prom any, millis int64) *Promise {
	base := el.Resolve(prom)
	return el.NewPromise(func(resolve, reject func(v any)) error {
		id := el.SetTimeout(func() {
			reject(NewTimeoutError("promise timed out"))
		}, millis)

		base.Then(func(v any) (any, error) {
			el.ClearTimeout(id)
			resolve(v)
			return nil, nil
		}, func(r error) (any, error) {
			el.ClearTimeout(id)
			reject(r)
			return nil, nil
		})
		return nil
	})
}

// SetInterval 模拟 setInterval 函数，以指定毫秒数为周期，重复调用回调函数
//   - callback 回调函数
//   - millis 延迟执行的毫秒数，自动修正负值的延迟为0
//
// # return
//
// 定时器 ID，可通过调用 [EventLoop.ClearInterval] 函数来清除定时器
func (el *EventLoop) SetInterval(callback func(), millis int64) int {
	if callback == nil {
		return -1
	}
	return el.timeline.produceTask(callback, millis, true)
}

// SetTimeout 模拟 setTimeout 函数，在指定毫秒数后调用回调函数
//   - callback 回调函数
//   - millis 延迟执行的毫秒数，自动修正负值的延迟为0
//
// # return
//
// 定时器 ID，可通过调用 [EventLoop.ClearTimeout] 函数来清除定时器
func (el *EventLoop) SetTimeout(callback func(), millis int64) int {
	if callback == nil {
		return -1
	}
	return el.timeline.produceTask(callback, millis, false)
}
