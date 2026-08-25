package promise_test

import (
	"sync/atomic"
	"testing"
	"time"
)

// TestSetInterval 覆盖 EventLoop.SetInterval 全部分支。
func TestSetInterval(t *testing.T) {
	t.Parallel()

	t.Run("fires", func(t *testing.T) {
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
	})

	t.Run("cancel-first", func(t *testing.T) {
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
	})

	t.Run("cancel-non-first", func(t *testing.T) {
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
	})

	t.Run("long-delay", func(t *testing.T) {
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
	})

	t.Run("nil-callback", func(t *testing.T) {
		t.Parallel()
		if got := el.SetInterval(nil, 100); got != -1 {
			t.Fatalf("SetInterval(nil) 应返回 -1，实际 %d", got)
		}
	})
}
