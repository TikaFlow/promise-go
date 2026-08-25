package promise

import (
	"sync"
	"time"
)

// invalidTimerID 无效定时器 ID：调度失败或事件循环已停止（队列关闭）时返回/传入。
const invalidTimerID = -1

// 时间线类型，用于调度宏任务
type timeLine struct {
	nextID    int
	idLock    sync.Mutex
	tasks     []*timedTask
	timer     *time.Timer
	taskCh    *Queue[*timedTask]
	clearCh   *Queue[int]
	eventLoop *EventLoop
}

// 定时任务类型
type timedTask struct {
	id       int
	deadline time.Time
	millis   int64
	callback func()
	repeat   bool
}

// 将任务推送到宏任务队列
func (tl *timeLine) queueMacrotask(fn func()) {
	tl.eventLoop.macrotaskQueue.Push(fn)
}

// 重置定时器，参数是到期时间
func (tl *timeLine) resetTimer(t time.Duration) {
	if !tl.timer.Stop() {
		select {
		case <-tl.timer.C:
		default:
		}
	}

	tl.timer.Reset(t)
}

// 添加一个待调度的任务
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

// 移除一个待调度的任务
func (tl *timeLine) removeTask(id int) {
	if id == invalidTimerID {
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

// 构造一个定时任务并添加到调度队列
func (tl *timeLine) produceTask(callback func(), millis int64, repeat bool) int {
	if callback == nil {
		return invalidTimerID
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

	if !tl.taskCh.Push(task) {
		return invalidTimerID
	}
	return id
}

// 消费任务，即把到达调度时间的任务推送到宏任务队列
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

// 调度器主逻辑
func (tl *timeLine) run() {
	for {
		select {
		case task, ok := <-tl.taskCh.Pop():
			if !ok {
				return
			}
			tl.appendTask(task)
		case id, ok := <-tl.clearCh.Pop():
			if !ok {
				return
			}
			tl.removeTask(id)
		case <-tl.timer.C:
			tl.consumeTask()
		case <-tl.eventLoop.done:
			tl.taskCh.Close()
			tl.clearCh.Close()
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
