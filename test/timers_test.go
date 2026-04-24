package promise_test

import (
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 测试SetInterval函数
func TestSetInterval(t *testing.T) {
	t.Parallel()
	var str string
	var count int
	var id int

	id = el.SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			el.ClearInterval(id)
		}
	}, 200)

	el.SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 180)

	el.SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 220)

	el.SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 380)

	el.SetTimeout(func() {
		if str != "interval interval " {
			t.Errorf("Expected str 'interval interval ', got %s", str)
		}
	}, 420)

	el.SetTimeout(func() {
		if str != "interval interval " {
			t.Errorf("Expected str 'interval interval ', got %s", str)
		}
	}, 580)

	el.SetTimeout(func() {
		if str != "interval interval interval " {
			t.Errorf("Expected str 'interval interval interval ', got %s", str)
		}
	}, 620)

	time.Sleep(time.Second)
}

// 测试SetInterval函数 - 取消 - 第1次执行
func TestSetIntervalCancelFirst(t *testing.T) {
	t.Parallel()
	var str string
	var count int
	var id int

	id = el.SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			el.ClearInterval(id)
		}
	}, 200)

	el.SetTimeout(func() {
		el.ClearInterval(id)
	}, 20)

	el.SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 240)

	time.Sleep(time.Second)
}

// 测试SetInterval函数 - 取消 - 非首次执行
func TestSetIntervalCancelNonFirst(t *testing.T) {
	t.Parallel()
	var str string
	var count int
	var id int

	id = el.SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			el.ClearInterval(id)
		}
	}, 200)

	el.SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
		el.ClearInterval(id)
	}, 220)

	el.SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 620)

	time.Sleep(time.Second)
}

// 测试SetInterval函数 - 长延迟
func TestSetIntervalLongDelay(t *testing.T) {
	t.Parallel()
	var str string
	var id int

	count := 0
	id = el.SetInterval(func() {
		str += "interval "
		count++
		if count >= 3 {
			el.ClearInterval(id)
		}
	}, 1000)

	el.SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 980)
	el.SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 1020)

	el.SetTimeout(func() {
		if str != "interval " {
			t.Errorf("Expected str 'interval ', got %s", str)
		}
	}, 1980)
	el.SetTimeout(func() {
		if str != "interval interval " {
			t.Errorf("Expected str 'interval interval ', got %s", str)
		}
	}, 2020)

	el.SetTimeout(func() {
		if str != "interval interval " {
			t.Errorf("Expected str 'interval interval ', got %s", str)
		}
	}, 2980)
	el.SetTimeout(func() {
		if str != "interval interval interval " {
			t.Errorf("Expected str 'interval interval interval ', got %s", str)
		}
	}, 3020)

	time.Sleep(5 * time.Second)
}

// 测试SetTimeout函数
func TestSetTimeout(t *testing.T) {
	t.Parallel()
	var str string
	el.SetTimeout(func() {
		str = "timeout value"
	}, 100)

	el.SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 80)

	el.SetTimeout(func() {
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	}, 120)

	time.Sleep(time.Second)
}

// 测试SetTimeout函数 - 取消
func TestSetTimeoutCancel(t *testing.T) {
	t.Parallel()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		id := el.SetTimeout(func() {
			resolve("timeout value")
		}, 100)
		el.ClearTimeout(id)
		return nil
	})

	el.SetTimeout(func() {
		if p.State() != Pending {
			t.Errorf("Expected state Pending, got %v", p.State())
		}
	}, 120)

	time.Sleep(time.Second)
}

// 测试SetTimeout函数 - 长延迟
func TestSetTimeoutLongDelay(t *testing.T) {
	t.Parallel()
	var str string
	el.SetTimeout(func() {
		str = "timeout value"
	}, 1000)

	el.SetTimeout(func() {
		if str != "" {
			t.Errorf("Expected str '', got %s", str)
		}
	}, 980)

	el.SetTimeout(func() {
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	}, 1020)

	time.Sleep(5 * time.Second)
}

// 测试SetTimeout函数 - 毫秒数为负数
func TestSetTimeoutNegativeMillis(t *testing.T) {
	t.Parallel()
	var str string
	el.SetTimeout(func() {
		str = "timeout value"
	}, -100)

	el.SetTimeout(func() {
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	}, 20)

	time.Sleep(time.Second)
}

// 测试SetTimeout函数 - 毫秒数为 0
func TestSetTimeoutZeroMillis(t *testing.T) {
	t.Parallel()
	var str string
	el.SetTimeout(func() {
		str = "timeout value"
	}, 0)

	el.SetTimeout(func() {
		if str != "timeout value" {
			t.Errorf("Expected str 'timeout value', got %s", str)
		}
	}, 0)

	time.Sleep(time.Second)
}
