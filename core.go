package promise

import (
	"time"

	pool "github.com/TikaFlow/worker-pool"
)

// StartEventLoop 启动一个事件循环
//
// 事件循环会持续运行，直到调用 [DocEventLoop.Stop] 方法关闭
//   - workerCount: 工作线程数量
func StartEventLoop(workerCount int) EventLoop {
	el := &eventLoopImpl{
		microtaskQueue: make(chan func(), 1024*10),
		macrotaskQueue: make(chan func(), 1024*10),
		done:           make(chan struct{}),
	}

	config := &pool.Config{
		BufferSize: 1024,
	}
	el.looper = pool.New(1, config)
	el.scheduler = pool.New(1, config)
	el.worker = pool.New(workerCount, nil)
	el.timeline = &timeLine{
		nextID:    0,
		tasks:     make([]*timedTask, 0, 1024*10),
		timer:     time.NewTimer(100 * 365 * 24 * time.Hour),
		taskCh:    make(chan *timedTask, 1024*10),
		clearCh:   make(chan int, 1024*10),
		eventLoop: el,
	}
	el.hooks = &promiseHooks{
		createdHookKeys:   make([]string, 0, 64),
		chainedHookKeys:   make([]string, 0, 64),
		fulfilledHookKeys: make([]string, 0, 64),
		rejectedHookKeys:  make([]string, 0, 64),
		settledHookKeys:   make([]string, 0, 64),
		hooks:             make(map[string]func(p Promise)),
	}

	el.looper.Add(el.run)
	el.scheduler.Add(el.timeline.run)

	return el
}
