package promise

import (
	"io"
	"sync"
	"time"
)

const (
	margin time.Duration = time.Millisecond * 128
	magic  time.Duration = 86413 * time.Second
)

var (
	microtaskQuene chan func()   = make(chan func(), 1024*10)
	originTimeout  time.Duration = time.Millisecond * 512
	timeout        time.Duration = originTimeout
	loopTimer      *time.Timer   = time.NewTimer(timeout + margin)
	done           chan struct{} = make(chan struct{})
	eventLoopID    uint64        = 0
	timeoutLock    sync.RWMutex
	loopTimerLock  sync.Mutex
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
		value:           nil,
		reason:          nil,
		state:           Pending,
		settledHandlers: make(chan *handler, 128),
		settled:         make(chan struct{}),
	}
	callHooks(PromiseCreated, prom)

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
QueueMicrotask 将回调函数添加到微任务队列末尾。
*/
func QueueMicrotask(fn func()) {
	microtaskQuene <- fn
	resetLoopTimer()
}

/*
Done 返回一个通道，当事件循环结束时，该通道会被关闭。
*/
func Done() <-chan struct{} {
	return done
}

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
					timeoutLock.RLock()
					if timeout == magic {
						timeoutLock.RUnlock()
						resetLoopTimer()
						continue loop
					}
					timeoutLock.RUnlock()
					close(done)
				}
			}
		}
	}
}
