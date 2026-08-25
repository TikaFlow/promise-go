package promise_test

import (
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestSetTimeout 覆盖 EventLoop.SetTimeout 全部分支。
func TestSetTimeout(t *testing.T) {
	t.Parallel()

	t.Run("fires-after-delay", func(t *testing.T) {
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
	})

	t.Run("cancel", func(t *testing.T) {
		t.Parallel()
		p := el.NewPromise(func(resolve, reject func(v any)) error {
			id := el.SetTimeout(func() {
				resolve("timeout value")
			}, 1000)
			time.Sleep(20 * time.Millisecond)
			el.ClearTimeout(id)
			return nil
		})
		time.Sleep(2 * time.Second)
		if p.State() != Pending {
			t.Fatalf("Expected state Pending, got %v", p.State())
		}
	})

	t.Run("long-delay", func(t *testing.T) {
		t.Parallel()
		var str string
		el.SetTimeout(func() {
			str = "timeout value"
		}, 1000)
		time.Sleep(3 * time.Second)
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	})

	t.Run("negative-millis", func(t *testing.T) {
		t.Parallel()
		var str string
		el.SetTimeout(func() {
			str = "timeout value"
		}, -100)
		time.Sleep(2 * time.Second)
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	})

	t.Run("zero-millis", func(t *testing.T) {
		t.Parallel()
		var str string
		el.SetTimeout(func() {
			str = "timeout value"
		}, 0)
		time.Sleep(2 * time.Second)
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	})

	t.Run("nil-callback", func(t *testing.T) {
		t.Parallel()
		if got := el.SetTimeout(nil, 100); got != -1 {
			t.Fatalf("SetTimeout(nil) 应返回 -1，实际 %d", got)
		}
	})
}

// TestTimerAPIAfterStop 验证 Stop 后调用定时器 API 不再 panic（send-on-close 已消除）。
// 跨定时器 API 行为，保留为独立测试函数。
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

// TestTimerPanicLoopContinues 验证宏任务（定时器回调）panic 不中断事件循环：后续定时器仍能执行。
// 跨定时器行为，保留为独立测试函数。
func TestTimerPanicLoopContinues(t *testing.T) {
	t.Parallel()
	el2 := StartEventLoop(1)
	defer el2.Stop()

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
