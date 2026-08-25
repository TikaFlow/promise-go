package promise_test

import (
	"sync/atomic"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 测试SetInterval函数
func TestSetInterval(t *testing.T) {
	t.Parallel()

	var (
		str   string
		count atomic.Int32
		id    int
	)

	id = el.SetInterval(func() {
		n := count.Add(1)
		switch n {
		case 1:
			str = "interval "
		case 2:
			str = "interval interval "
		case 3:
			str = "interval interval interval "
			el.ClearInterval(id)
		}
	}, 200)

	time.Sleep(3 * time.Second)
	if str != "interval interval interval " {
		t.Errorf("Expected str 'interval interval interval ', got %s", str)
	}
}

// 测试SetInterval函数 - 取消 - 第1次执行
func TestSetIntervalCancelFirst(t *testing.T) {
	t.Parallel()

	var (
		count atomic.Int32
		id    int
	)

	id = el.SetInterval(func() {
		count.Add(1)
	}, 200)

	el.SetTimeout(func() {
		el.ClearInterval(id)
	}, 20)

	time.Sleep(2 * time.Second)
	if n := count.Load(); n != 0 {
		t.Errorf("Expected count 0 (cancel-first), got %d", n)
	}
}

// 测试SetInterval函数 - 取消 - 非首次执行
func TestSetIntervalCancelNonFirst(t *testing.T) {
	t.Parallel()

	var (
		count atomic.Int32
		id    int
	)

	id = el.SetInterval(func() {
		count.Add(1)
	}, 200)

	// 220ms：第一次 tick 已触发（~200ms），之后立即取消
	el.SetTimeout(func() {
		el.ClearInterval(id)
	}, 220)

	time.Sleep(2 * time.Second)

	if n := count.Load(); n != 1 {
		t.Errorf("Expected count 1 (cancel-non-first), got %d", n)
	}
}

// 测试SetInterval函数 - 长延迟
func TestSetIntervalLongDelay(t *testing.T) {
	t.Parallel()

	var (
		count atomic.Int32
		str   string
		id    int
	)

	id = el.SetInterval(func() {
		n := count.Add(1)
		switch n {
		case 1:
			str = "interval "
		case 2:
			str = "interval interval "
		case 3:
			str = "interval interval interval "
			el.ClearInterval(id)
		}
	}, 1000)

	time.Sleep(5 * time.Second)

	if str != "interval interval interval " {
		t.Errorf("Expected str 'interval interval interval ', got '%s'", str)
	}
	if n := count.Load(); n != 3 {
		t.Errorf("Expected count 3 (no extra tick), got %d", n)
	}
}

// 测试SetTimeout函数
func TestSetTimeout(t *testing.T) {
	t.Parallel()

	var str string

	el.SetTimeout(func() {
		str = "timeout value"
	}, 1000)

	// 20ms：未触发
	el.SetTimeout(func() {
		if s := str; s != "" {
			t.Errorf("Expected empty str, got '%s'", s)
		}
	}, 20)

	time.Sleep(3 * time.Second)

	if s := str; s != "timeout value" {
		t.Errorf("Expected str 'timeout value', got '%s'", s)
	}
}

// 测试SetTimeout函数 - 取消
func TestSetTimeoutCancel(t *testing.T) {
	t.Parallel()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		id := el.SetTimeout(func() {
			resolve("timeout value")
		}, 1000)
		time.Sleep(20 * time.Millisecond)
		el.ClearTimeout(id)
		return nil
	})

	time.Sleep(3 * time.Second)

	if p.State() != Pending {
		t.Errorf("Expected state Pending, got %v", p.State())
	}
}

// 测试SetTimeout函数 - 长延迟
func TestSetTimeoutLongDelay(t *testing.T) {
	t.Parallel()

	var str string

	el.SetTimeout(func() {
		str = "timeout value"
	}, 1000)

	time.Sleep(3 * time.Second)

	if str != "timeout value" {
		t.Errorf("Expected str 'timeout value', got %s", str)
	}
}

// 测试SetTimeout函数 - 毫秒数为负数
func TestSetTimeoutNegativeMillis(t *testing.T) {
	t.Parallel()

	var str string

	el.SetTimeout(func() {
		str = "timeout value"
	}, -100)

	time.Sleep(2 * time.Second)

	if str != "timeout value" {
		t.Errorf("Expected str 'timeout value', got %s", str)
	}
}

// 测试SetTimeout函数 - 毫秒数为 0
func TestSetTimeoutZeroMillis(t *testing.T) {
	t.Parallel()

	var str string

	el.SetTimeout(func() {
		str = "timeout value"
	}, 0)

	time.Sleep(2 * time.Second)

	if str != "timeout value" {
		t.Errorf("Expected str 'timeout value', got %s", str)
	}
}

// 测试 Stop 后调用定时器 API 不再 panic（send-on-close 已消除）。
func TestTimerAPIAfterStop(t *testing.T) {
	t.Parallel()
	el2 := StartEventLoop(1)
	id := el2.SetTimeout(func() {}, 10)
	el2.Stop()

	if got := el2.SetTimeout(func() {}, 10); got != -1 {
		t.Errorf("Stop 后 SetTimeout 应返回 -1，实际 %d", got)
	}
	if got := el2.SetInterval(func() {}, 10); got != -1 {
		t.Errorf("Stop 后 SetInterval 应返回 -1，实际 %d", got)
	}
	el2.ClearTimeout(id) // 有效 id，应在队列关闭后安全丢弃
	el2.ClearInterval(id)
	el2.ClearTimeout(-1) // 无效 id，直接返回
}

// 测试宏任务（定时器回调）panic 不中断事件循环：后续定时器仍能执行。
func TestTimerPanicLoopContinues(t *testing.T) {
	t.Parallel()
	el2 := StartEventLoop(1)
	defer el2.Stop()

	// 第一个定时器回调 panic，第二个应仍正常触发。
	el2.SetTimeout(func() {
		panic("boom")
	}, 10)

	done := make(chan struct{})
	el2.SetTimeout(func() {
		close(done)
	}, 100)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("宏任务 panic 带崩了事件循环，后续定时器未执行")
	}
}
