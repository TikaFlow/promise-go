package promise

import (
	"errors"
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	margin time.Duration = time.Millisecond * 128
	magic  time.Duration = 86413 * time.Second
)

var (
	taskQuene      chan func()   = make(chan func(), 1024*10)
	microtaskQuene chan func()   = make(chan func(), 1024*10)
	originTimeout  time.Duration = time.Millisecond * 512
	timeout        time.Duration = originTimeout
	loopTimer      *time.Timer   = time.NewTimer(timeout + margin)
	done           chan struct{} = make(chan struct{})
	eventLoopID    uint64        = 0
	timeoutLock    sync.Mutex
	timerLock      sync.Mutex
)

func init() {
	go timerScheduler()
	go startEventLoop()
}

type eventLoopHandler struct {
	doneCh chan struct{}
}

/*
[io.Closer.Close]
*/
func (elh *eventLoopHandler) Close() error {
	close(elh.doneCh)
	return nil
}

/*
EventLoopHandler 获取事件循环句柄，用于关闭事件循环。

注意：一旦获取了事件循环句柄，就必须使用其 Close 方法关闭事件循环，否则会导致事件循环无法退出。
*/
func EventLoopHandler() io.Closer {
	timeoutLock.Lock()
	timeout = magic
	originTimeout = magic
	timeoutLock.Unlock()
	return &eventLoopHandler{
		doneCh: done,
	}
}

/*
New 创建一个新的 Promise 实例。
  - exec 执行器函数，用于定义 Promise 的异步操作。
*/
func New(exec Executor) Promise {
	if exec == nil {
		panic("Promise executor must be a function")
	}

	prom := &promiseImpl{
		state:           Pending,
		result:          nil,
		settledHandlers: make(chan *handler, 128),
		settled:         make(chan struct{}),
	}

	res := func(data any) {
		prom.resolved.Do(func() {
			resolvePromise(prom, data)
		})
	}
	rej := func(reason any) {
		prom.resolved.Do(func() {
			rejectPromsie(prom, reason)
		})
	}

	if err := exec(res, rej); err != nil {
		rej(err)
	}
	return prom
}

/*
All 等待所有 Promise 解决。
  - 如果所有 Promise 都成功解决，新 Promise 也会成功解决，且解决值为一个包含所有 Promise 解决值的数组；
  - 如果任何一个 Promise 被拒绝，新 Promise 也会被拒绝，且拒绝理由为第一个被拒绝的 Promise 的拒绝理由。
*/
func All(proms ...any) Promise {
	if proms == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(proms) == 0 {
		return Resolve(make([]any, 0))
	}

	return New(func(resolve, reject func(v any)) error {
		results := make([]any, len(proms))
		var count int32 = 0

		for index, item := range proms {
			prom := Resolve(item)
			prom.Then(func(v any) (any, error) {
				results[index] = v
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) {
					resolve(results)
				}
				return nil, nil
			}, func(reason any) (any, error) {
				reject(reason)
				return nil, nil
			})
		}

		return nil
	})
}

/*
AllSettled 等待所有 Promise 完成（无论成功失败）。
  - 新 Promise 会在所有 Promise 完成后解决，解决值为一个包含所有 Promise 完成状态和结果的数组。
*/
func AllSettled(proms ...any) Promise {
	if proms == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(proms) == 0 {
		return Resolve(make([]map[string]any, 0))
	}

	return New(func(resolve, reject func(v any)) error {
		type result struct {
			Status string
			Value  any
			Reason any
		}

		results := make([]result, len(proms))
		var count int32 = 0
		for index, item := range proms {
			prom := Resolve(item)
			settleData := func() {
				finalResults := make([]map[string]any, len(results))
				for i, r := range results {
					finalResults[i] = make(map[string]any)
					finalResults[i]["status"] = r.Status
					if r.Status == Fulfilled {
						finalResults[i]["value"] = r.Value
					} else {
						finalResults[i]["reason"] = r.Reason
					}
				}
				resolve(finalResults)
			}
			prom.Then(func(v any) (any, error) {
				results[index] = result{Status: Fulfilled, Value: v}
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) {
					settleData()
				}
				return nil, nil
			}, func(reason any) (any, error) {
				results[index] = result{Status: Rejected, Reason: reason}
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) {
					settleData()
				}
				return nil, nil
			})
		}

		return nil
	})
}

/*
Any 等待第一个 Promise 解决。
  - 如果任何一个 Promise 解决，新 Promise 也会被解决，且解决值为第一个被解决的 Promise 的解决值。
  - 如果所有 Promise 都被拒绝，新 Promise 也会被拒绝，且拒绝理由为一个包含所有 Promise 拒绝理由的 map。
*/
func Any(proms ...any) Promise {
	if proms == nil {
		return Reject("TypeError: nil is not iterable")
	}
	if len(proms) == 0 {
		result := make(map[string]any)
		result["errors"] = make([]any, 0)
		result["stack"] = "AggregateError: All promises were rejected"
		result["message"] = "All promises were rejected"
		return Reject(result)
	}

	return New(func(resolve, reject func(v any)) error {
		reasons := make([]any, len(proms))
		var count int32 = 0

		for index, item := range proms {
			prom := Resolve(item)
			prom.Then(func(v any) (any, error) {
				resolve(v)
				return nil, nil
			}, func(reason any) (any, error) {
				reasons[index] = reason
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(proms) {
					result := make(map[string]any)
					result["errors"] = reasons
					result["stack"] = "AggregateError: All promises were rejected"
					result["message"] = "All promises were rejected"
					reject(result)
				}
				return nil, nil
			})
		}

		return nil
	})
}

/*
Race 等待第一个 Promise 完成。
  - 新 Promise 会在第一个 Promise 完成后解决或拒绝，解决值或拒绝理由跟随第一个完成的 Promise。
*/
func Race(proms ...any) Promise {
	if proms == nil {
		return Reject("TypeError: nil is not iterable")
	}

	return New(func(resolve, reject func(v any)) error {
		if len(proms) == 0 {
			return nil
		}

		for _, item := range proms {
			prom := Resolve(item)
			prom.Then(func(v any) (any, error) {
				resolve(v)
				return nil, nil
			}, func(reason any) (any, error) {
				reject(reason)
				return nil, nil
			})
		}

		return nil
	})
}

/*
Resolve 返回一个已解决的 Promise，解决值为指定值 value。

如果 value 已经是 Promise，则直接返回该 Promise，详见 [MDN]。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/resolve
*/
func Resolve(value any) Promise {
	if prom, ok := value.(Promise); ok {
		return prom
	}

	return New(func(resolve, reject func(v any)) error {
		resolve(value)
		return nil
	})
}

/*
Reject 返回一个已拒绝的 Promise，拒绝理由为指定值，详见 [MDN]。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/reject
*/
func Reject(reason any) Promise {
	return New(func(resolve, reject func(v any)) error {
		reject(reason)
		return nil
	})
}

/*
Try 接受一个任意类型的回调函数（无论其是同步或异步，返回结果或抛出异常），并将其结果封装成一个 Promise，详见 [MDN]。
  - fn 任意类型的回调函数，接受任意数量的参数，函数返回值格式与 [ThenCallback] 相同。
  - args 将要传递给 fn 函数的参数列表。

返回一个 Promise，其状态可以是：
  - 已解决（Fulfilled）：如果 fn 函数返回一个非错误值。
  - 已拒绝（Rejected）：如果 fn 函数返回一个错误值。
  - 异步解决或拒绝：如果 fn 函数返回一个 Promise，新 Promise 会吸收该 Promise 的状态。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/try
*/
func Try(fn func(...any) (any, error), args ...any) Promise {
	return New(func(resolve, reject func(v any)) error {
		if fn == nil {
			reject("Promise executor must be a function")
			return nil
		}

		result, err := fn(args...)
		if err != nil {
			reject(result)
			return nil
		}
		resolve(result)
		return nil
	})
}

/*
PromiseWithResolvers 创建一个新的 Promise 实例，同时返回 resolve 和 reject 函数，
对应于传入给 Promise() 构造函数执行器的两个参数。

这使得可以在 Promise 外部手动解决或拒绝 Promise，详见 [MDN]。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/withResolvers#%E6%8F%8F%E8%BF%B0
*/
func PromiseWithResolvers() (Promise, func(any), func(any)) {
	var resolve, reject func(any)
	p := New(func(res func(any), rej func(any)) error {
		resolve = res
		reject = rej
		return nil
	})
	return p, resolve, reject
}

/*
QueueMicrotask 将回调函数添加到微任务队列末尾。
*/
func QueueMicrotask(fn func()) {
	microtaskQuene <- fn
	resetLoopTimer()
}

/*
Async 将代码作为一个异步任务执行。

这是一个语法糖，等价于以下语句：

	SetTimeout(fn, 0)

使用此函数包裹同步代码后，将完美模拟事件循环中 Promise 的行为，详见 [MDN]。

[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Execution_model#%E4%BD%9C%E4%B8%9A%E9%98%9F%E5%88%97%E4%B8%8E%E4%BA%8B%E4%BB%B6%E5%BE%AA%E7%8E%AF
*/
func Async(fn func()) {
	SetTimeout(fn, 0)
}

/*
Await 等待 Promise 完成，并设定超时时间，以免无限等待。

  - prom 需要等待的 Promise 实例，如果不是 Promise 实例，则会被包装成 Promise。
  - timeout 超时时间，单位为毫秒。

返回值：与 [ThenCallback] 相同，v 总是需要的值（包括错误信息），而 err 仅代表是否出错。

v 的值可能是：

  - 如果 Promise 在超时时间内未完成，则为超时错误。
  - 如果 Promise 成功，v 为 Promise 解决值。
  - 如果 Promise 拒绝，v 为 Promise 拒绝理由。
*/
func Await(prom any, timeout int64) (v any, err error) {
	negErr := errors.New("await timeout must be greater than 0")
	if timeout <= 0 {
		return negErr, negErr
	}

	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	defer timer.Stop()

	timeoutErr := errors.New("TimeoutError: await timeout")
	prom2 := Resolve(prom)
	select {
	case <-prom2.Done():
		v = prom2.Result()
		if prom2.State() == Rejected {
			err = errors.New("promise rejected")
		}
	case <-timer.C:
		v, err = timeoutErr, timeoutErr
	}

	return
}

/*
Delay 返回一个新的 Promise，其状态会在延迟时间后被解决。

  - prom 将会使用的已决值，如果 prom 是 Promise 实例，则会等待其完成后才开始计时；
    如果是一个已拒绝的 Promise，则会立即拒绝新 Promise。
  - timeout 延迟时间，单位为毫秒。
*/
func Delay(prom any, millis int64) Promise {
	return New(func(resolve, reject func(v any)) error {
		Resolve(prom).Then(func(v2 any) (any, error) {
			go func() {
				time.Sleep(time.Duration(millis) * time.Millisecond)
				resolve(v2)
			}()
			return nil, nil
		}, func(r any) (any, error) {
			reject(r)
			return nil, nil
		})
		return nil
	})
}

/*
Done 返回一个通道，当事件循环结束时，该通道会被关闭。
*/
func Done() <-chan struct{} {
	return done
}

// ====== 辅助函数 ======
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

	if prom.state != Pending {
		return
	}

	if value == prom {
		// 2.3.1 如果 Promise 和已决值相同，则抛出 TypeError 异常
		rejectPromsie(prom, "TypeError: Chaining cycle detected for promise")
		return
	}

	// 2.3.2
	if x, ok := value.(Promise); ok {
		// 2.3.2 如果已决值是 Promise 对象，则采用其状态
		QueueMicrotask(func() {
			x.Then(func(v any) (any, error) {
				resolvePromise(prom, v)
				return nil, nil
			}, func(r any) (any, error) {
				rejectPromsie(prom, r)
				return nil, nil
			})
		})
		return
	}
	// 2.3.3 同上

	// 2.3.4 其他情况，则使用 value 作为已决值
	prom.state = Fulfilled
	prom.result = value
	close(prom.settled)
	flushHandlers(prom)
}

func rejectPromsie(prom *promiseImpl, reason any) {
	if getGoroutineID() != eventLoopID {
		SetTimeout(func() {
			rejectPromsie(prom, reason)
		}, 0)
		return
	}

	if prom == nil {
		return
	}

	if prom.state != Pending {
		return
	}

	prom.state = Rejected
	prom.result = reason
	close(prom.settled)
	flushHandlers(prom)
}

func queueTask(fn func()) {
	taskQuene <- fn
	resetLoopTimer()
}

func resetLoopTimer() {
	timerLock.Lock()
	defer timerLock.Unlock()

	if !loopTimer.Stop() {
		select {
		case <-loopTimer.C:
		default:
		}
	}
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
				if cur.state == Fulfilled {
					if hdl.onFulfilled == nil {
						// 2.2.1 如果回调不是函数，则忽略（穿透->2.2.7.3）
						resolvePromise(hdl.prom, cur.result)
						return
					} else {
						// 2.2.2
						res, err = hdl.onFulfilled(cur.result)
						if err != nil {
							// 2.2.7.2 如果回调函数抛出一个异常 e，则新 Promise 实例必须被拒绝，且拒绝原因为 e
							rejectPromsie(hdl.prom, res)
							return
						}
					}
				} else { // 必然是 Rejected
					if hdl.onRejected == nil {
						// 2.2.1 如果回调不是函数，则忽略（穿透->2.2.7.4）
						rejectPromsie(hdl.prom, cur.result)
						return
					} else {
						// 2.2.3
						res, err = hdl.onRejected(cur.result)
						if err != nil {
							// 2.2.7.2 如果回调函数抛出一个异常 e，则新 Promise 实例必须被拒绝，且拒绝原因为 e
							rejectPromsie(hdl.prom, res)
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

// ====== 辅助函数【结束】 ======

/*
startEventLoop 模拟事件循环：清空微队列 -> 执行一个宏任务（如有） -> 清空微队列 ...
*/
func startEventLoop() {
	eventLoopID = getGoroutineID()
	defer close(schedulerDone)
loop:
	for {
		select {
		case mtask := <-microtaskQuene:
			resetLoopTimer()
			mtask()
			resetLoopTimer()
		default:
			select {
			case task := <-taskQuene:
				resetLoopTimer()
				task()
				resetLoopTimer()
			default:
				select {
				case mtask := <-microtaskQuene:
					resetLoopTimer()
					mtask()
					resetLoopTimer()
				case task := <-taskQuene:
					resetLoopTimer()
					task()
					resetLoopTimer()
				case <-done:
					break loop
				case <-loopTimer.C:
					if timeout == magic {
						resetLoopTimer()
						continue loop
					}
					close(done)
				}
			}
		}
	}
}
