package promise

import (
	"sync"
	"time"
)

const (
	minDelay int64 = 1
)

var (
	nextID         int             = 1
	tasks          []*timerTask    = make([]*timerTask, 0, 1024*10)
	schedulerTimer *time.Timer     = time.NewTimer(magic)
	curTimeout     time.Duration   = magic
	taskCh         chan *timerTask = make(chan *timerTask, 1024)
	clearCh        chan int        = make(chan int, 1024)
	schedulerDone  chan struct{}   = make(chan struct{})
	idLock         sync.Mutex
)

type timerTask struct {
	id       int
	deadline time.Time
	millis   int64
	callback func()
	repeat   bool
}

func resetSchedulerTimer(t time.Duration) {
	if !schedulerTimer.Stop() {
		select {
		case <-schedulerTimer.C:
		default:
		}
	}
	curTimeout = t
	schedulerTimer.Reset(t)
}

func appendTask(task *timerTask) {
	if task.millis > timeout.Milliseconds() {
		timeoutLock.Lock()
		timeout = time.Duration(task.millis) * time.Millisecond
		timeoutLock.Unlock()
		resetLoopTimer()
	}

	if len(tasks) == 0 {
		tasks = append(tasks, task)
		resetSchedulerTimer(time.Until(tasks[0].deadline))
		return
	}

	low, high := 0, len(tasks)-1
	for low <= high {
		mid := (low + high) / 2
		if tasks[mid].deadline.After(task.deadline) {
			high = mid - 1
		} else {
			low = mid + 1
		}
	}

	tasks = append(tasks, nil)
	copy(tasks[low+1:], tasks[low:])
	tasks[low] = task

	if low == 0 {
		resetSchedulerTimer(time.Until(tasks[0].deadline))
	}
}

func removeTask(id int) {
	if id == -1 {
		return
	}

	for i, task := range tasks {
		if task.id == id {
			tasks = append(tasks[:i], tasks[i+1:]...)

			if len(tasks) > 0 {
				resetSchedulerTimer(time.Until(tasks[0].deadline))
			} else {
				resetSchedulerTimer(magic)
			}
			return
		}
	}
}

func produceTask(callback func(), millis int64, repeat bool) int {
	if callback == nil {
		return -1
	}

	if millis < minDelay {
		millis = minDelay
	}

	idLock.Lock()
	defer idLock.Unlock()

	id := nextID
	nextID++
	task := &timerTask{
		id:       id,
		deadline: time.Now().Add(time.Duration(millis) * time.Millisecond),
		millis:   millis,
		callback: callback,
		repeat:   repeat,
	}
	taskCh <- task
	return id
}

func comsumeTask() {
	if len(tasks) > 0 && !tasks[0].deadline.After(time.Now()) {
		task := tasks[0]
		tasks = tasks[1:]

		if len(tasks) == 0 {
			timeoutLock.Lock()
			timeout = originTimeout
			timeoutLock.Unlock()
		}
		queueTask(task.callback)

		if task.repeat {
			task.deadline = time.Now().Add(time.Duration(task.millis) * time.Millisecond)
			appendTask(task)
		}

		if len(tasks) > 0 {
			resetSchedulerTimer(time.Until(tasks[0].deadline))
		} else {
			resetSchedulerTimer(magic)
		}
	}
}

/*
SetTimeout 模拟 setTimeout 函数，在指定毫秒数后调用回调函数。
  - callback 回调函数
  - millis 延迟执行的毫秒数，具有最低延迟限制：1ms

返回一个定时器 ID，可通过调用 ClearTimeout 函数来清除定时器。
*/
func SetTimeout(callback func(), millis int64) int {
	return produceTask(callback, millis, false)
}

/*
SetInterval 模拟 setInterval 函数，在指定毫秒数后重复调用回调函数。
  - callback 回调函数
  - millis 延迟执行的毫秒数，具有最低延迟限制：1ms

返回一个定时器 ID，可通过调用 ClearInterval 函数来清除定时器。
*/
func SetInterval(callback func(), millis int64) int {
	return produceTask(callback, millis, true)
}

/*
ClearTimeout 清除由 SetTimeout 函数创建的定时器。
  - id 定时器ID
*/
func ClearTimeout(id int) {
	clearCh <- id
}

/*
ClearInterval 清除由 SetInterval 函数创建的定时器。
  - id 定时器ID
*/
func ClearInterval(id int) {
	clearCh <- id
}

func timerScheduler() {
	for {
		select {
		case task := <-taskCh:
			appendTask(task)
		case id := <-clearCh:
			removeTask(id)
		case <-schedulerTimer.C:
			if curTimeout == magic {
				resetSchedulerTimer(magic)
				continue
			}
			comsumeTask()
		case <-schedulerDone:
			return
		}
	}
}
