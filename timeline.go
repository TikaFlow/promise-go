package promise

import (
	"sync"
	"time"
)

type timeLine struct {
	nextID    int
	idLock    sync.Mutex
	tasks     []*timedTask
	timer     *time.Timer
	taskCh    chan *timedTask
	clearCh   chan int
	eventLoop *eventLoopImpl
}

type timedTask struct {
	id       int
	deadline time.Time
	millis   int64
	callback func()
	repeat   bool
}

func (tl *timeLine) queueMacrotask(fn func()) {
	tl.eventLoop.macrotaskQueue <- fn
}

func (tl *timeLine) resetTimer(t time.Duration) {
	if !tl.timer.Stop() {
		select {
		case <-tl.timer.C:
		default:
		}
	}

	tl.timer.Reset(t)
}

func (tl *timeLine) appendTask(task *timedTask) {
	if len(tl.tasks) == 0 {
		tl.tasks = append(tl.tasks, task)
		tl.resetTimer(time.Until(tl.tasks[0].deadline))
		return
	}

	low, high := 0, len(tl.tasks)-1
	for low <= high {
		mid := (low + high) / 2
		if tl.tasks[mid].deadline.After(task.deadline) {
			high = mid - 1
		} else {
			low = mid + 1
		}
	}

	tl.tasks = append(tl.tasks, nil)
	copy(tl.tasks[low+1:], tl.tasks[low:])
	tl.tasks[low] = task

	if low == 0 {
		tl.resetTimer(time.Until(tl.tasks[0].deadline))
	}
}

func (tl *timeLine) removeTask(id int) {
	if id == -1 {
		return
	}

	for i, task := range tl.tasks {
		if task.id == id {
			tl.tasks = append(tl.tasks[:i], tl.tasks[i+1:]...)

			if len(tl.tasks) > 0 {
				tl.resetTimer(time.Until(tl.tasks[0].deadline))
			}
			return
		}
	}
}

func (tl *timeLine) produceTask(callback func(), millis int64, repeat bool) int {
	if callback == nil {
		return -1
	}

	if millis < 0 {
		millis = 0
	}

	tl.idLock.Lock()
	defer tl.idLock.Unlock()

	tl.nextID++
	id := tl.nextID
	task := &timedTask{
		id:       id,
		deadline: time.Now().Add(time.Duration(millis) * time.Millisecond),
		millis:   millis,
		callback: callback,
		repeat:   repeat,
	}
	tl.taskCh <- task
	return id
}

func (tl *timeLine) consumeTask() {
	called := false
	for len(tl.tasks) > 0 && !tl.tasks[0].deadline.After(time.Now()) {
		task := tl.tasks[0]
		tl.tasks = tl.tasks[1:]

		tl.queueMacrotask(task.callback)
		called = true

		if task.repeat {
			task.deadline = time.Now().Add(time.Duration(task.millis) * time.Millisecond)
			tl.appendTask(task)
		}
	}

	if len(tl.tasks) > 0 && called {
		tl.resetTimer(time.Until(tl.tasks[0].deadline))
	}
}

/*
SetTimeout 模拟 setTimeout 函数，在指定毫秒数后调用回调函数。
  - callback 回调函数
  - millis 延迟执行的毫秒数，具有最低延迟限制：1ms

返回一个定时器 ID，可通过调用 ClearTimeout 函数来清除定时器。
*/
func (el *eventLoopImpl) SetTimeout(callback func(), millis int64) int {
	if callback == nil {
		return -1
	}
	return el.timeline.produceTask(callback, millis, false)
}

/*
SetInterval 模拟 setInterval 函数，在指定毫秒数后重复调用回调函数。
  - callback 回调函数
  - millis 延迟执行的毫秒数，具有最低延迟限制：1ms

返回一个定时器 ID，可通过调用 ClearInterval 函数来清除定时器。
*/
func (el *eventLoopImpl) SetInterval(callback func(), millis int64) int {
	if callback == nil {
		return -1
	}
	return el.timeline.produceTask(callback, millis, true)
}

/*
ClearTimeout 清除由 SetTimeout 函数创建的定时器。
  - id 定时器ID
*/
func (el *eventLoopImpl) ClearTimeout(id int) {
	if id == -1 {
		return
	}
	el.timeline.clearCh <- id
}

/*
ClearInterval 清除由 SetInterval 函数创建的定时器。
  - id 定时器ID
*/
func (el *eventLoopImpl) ClearInterval(id int) {
	if id == -1 {
		return
	}
	el.timeline.clearCh <- id
}

func (tl *timeLine) run() {
	for {
		select {
		case task := <-tl.taskCh:
			tl.appendTask(task)
		case id := <-tl.clearCh:
			tl.removeTask(id)
		case <-tl.timer.C:
			tl.consumeTask()
		case <-tl.eventLoop.done:
			close(tl.taskCh)
			close(tl.clearCh)
			// 关闭定时器
			if !tl.timer.Stop() {
				select {
				case <-tl.timer.C:
				default:
				}
			}
			return
		}
	}
}
