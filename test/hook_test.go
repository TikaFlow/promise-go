package promise_test

import (
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

/*
为了钩子不被其他测试影响，使用单独的事件循环
*/

// 测试On方法
func TestOn(t *testing.T) {
	t.Parallel()

	el2 := StartEventLoop(1)
	defer el2.Stop()

	result := ""
	_ = el2.OnPromise(PromiseCreated, func(p *Promise) {
		result += "created-"
	})
	_ = el2.OnPromise(PromiseSettled, func(p *Promise) {
		result += "settled-"
	})
	_ = el2.OnPromise(PromiseFulfilled, func(p *Promise) {
		result += "fulfilled-"
	})
	_ = el2.OnPromise(PromiseRejected, func(p *Promise) {
		result += "rejected-"
	})
	_ = el2.OnPromise(PromiseChained, func(p *Promise) {
		result += "chained-"
	})

	p, res, _ := el2.PromiseWithResolvers()
	_, _, rej := el2.PromiseWithResolvers()
	// created
	expected := "created-created-"
	if result != expected {
		t.Errorf("result should be '%s', but got '%s'", expected, result)
	}

	// 	settled+rejected
	rej("error")
	time.Sleep(time.Second)
	expected += "settled-rejected-"
	if result != expected {
		t.Errorf("result should be '%s', but got '%s'", expected, result)
	}

	// 	created+chained
	p.Then(nil, nil)
	expected += "created-chained-"
	if result != expected {
		t.Errorf("result should be '%s', but got '%s'", expected, result)
	}

	// 	settled+fulfilled AND settled+fulfilled
	res("ok")
	time.Sleep(time.Second)
	expected += "settled-fulfilled-settled-fulfilled-"
	if result != expected {
		t.Errorf("result should be '%s', but got '%s'", expected, result)
	}
}

// 测试Off方法 - 成功解绑
func TestOffSuccess(t *testing.T) {
	t.Parallel()
	el2 := StartEventLoop(1)
	defer el2.Stop()

	called := false
	key := el2.OnPromise(PromiseCreated, func(p *Promise) {
		called = true
	})

	result := el2.OffPromise(PromiseCreated, key)
	if !result {
		t.Errorf("Expected Off to return true")
	}

	el2.NewPromise(func(resolve, reject func(v any)) error {
		resolve("test")
		return nil
	})

	time.Sleep(time.Second)
	if called {
		t.Errorf("Expected hook to not be called after Off")
	}
}

// 测试Off - 不匹配的key/hookType
func TestOffMismatch(t *testing.T) {
	t.Parallel()
	el2 := StartEventLoop(1)
	defer el2.Stop()

	key := el2.OnPromise(PromiseCreated, func(p *Promise) {})
	result := el2.OffPromise(PromiseRejected, key)
	if result {
		t.Errorf("Expected Off to return false for mismatched event type")
	}

	key = el2.OnPromise(PromiseChained, func(p *Promise) {})
	result = el2.OffPromise(PromiseChained, "this-is-a-key")
	if result {
		t.Errorf("Expected Off to return false for mismatched key")
	}
}

// 测试 panic 钩子：定时器回调 panic 应触发 AllPanic 与 TimerPanic，且 AllPanic 先触发。
func TestOnPanicTimer(t *testing.T) {
	t.Parallel()
	el2 := StartEventLoop(1)
	defer el2.Stop()

	var order []string
	el2.OnPanic(AllPanic, func(r any) {
		order = append(order, "all")
	})
	el2.OnPanic(TimerPanic, func(r any) {
		order = append(order, "timer")
	})

	el2.SetTimeout(func() {
		panic("boom")
	}, 10)
	time.Sleep(300 * time.Millisecond)

	if len(order) != 2 || order[0] != "all" || order[1] != "timer" {
		t.Errorf("期望先 AllPanic 后 TimerPanic，实际 %v", order)
	}
}

// 测试 panic 钩子：Promise 回调 panic 应触发 PromisePanic，且 Promise 被拒绝。
func TestOnPanicPromise(t *testing.T) {
	t.Parallel()
	el2 := StartEventLoop(1)
	defer el2.Stop()

	var promPanicCalled bool
	el2.OnPanic(PromisePanic, func(r any) {
		promPanicCalled = true
	})

	p := el2.Resolve("x").Then(func(v any) (any, error) {
		panic("boom")
	}, nil)
	mustSettle(t, p, 2*time.Second)

	if p.State() != Rejected {
		t.Errorf("Promise 应被拒绝")
	}
	if !promPanicCalled {
		t.Errorf("PromisePanic 钩子应被触发")
	}
}

// 测试 panic 钩子：Async 任务 panic 应触发 AsyncPanic，且 Promise 被拒绝。
func TestOnPanicAsync(t *testing.T) {
	t.Parallel()
	el2 := StartEventLoop(1)
	defer el2.Stop()

	var asyncPanicCalled bool
	el2.OnPanic(AsyncPanic, func(r any) {
		asyncPanicCalled = true
	})

	p := el2.Async(func() (any, error) {
		panic("boom")
	})
	mustSettle(t, p, 2*time.Second)

	if p.State() != Rejected {
		t.Errorf("Promise 应被拒绝")
	}
	if !asyncPanicCalled {
		t.Errorf("AsyncPanic 钩子应被触发")
	}
}

// 测试 panic 钩子解绑：OffPanic 后不再触发。
func TestOffPanic(t *testing.T) {
	t.Parallel()
	el2 := StartEventLoop(1)
	defer el2.Stop()

	called := false
	key := el2.OnPanic(TimerPanic, func(r any) {
		called = true
	})
	if !el2.OffPanic(TimerPanic, key) {
		t.Errorf("OffPanic 应返回 true")
	}

	el2.SetTimeout(func() {
		panic("boom")
	}, 10)
	time.Sleep(300 * time.Millisecond)

	if called {
		t.Errorf("解绑后 panic 钩子不应被触发")
	}
}

// 测试 HookPanic 级联与止啸叫：AllPanic 钩子 panic 触发 HookPanic，
// HookPanic 钩子自身 panic 应被静默吞掉（不再级联）。
func TestHookPanicCascade(t *testing.T) {
	t.Parallel()
	el2 := StartEventLoop(1)
	defer el2.Stop()

	var hookPanicCalled int
	el2.OnPanic(HookPanic, func(r any) {
		hookPanicCalled++
		panic("from-hook-panic-hook") // 应被静默，不再级联
	})

	var allCalls int
	el2.OnPanic(AllPanic, func(r any) {
		allCalls++
		if r == "trigger" {
			panic("from-all-panic-hook") // 仅首次触发 panic
		}
	})

	el2.SetTimeout(func() {
		panic("trigger")
	}, 10)
	time.Sleep(300 * time.Millisecond)

	if hookPanicCalled == 0 {
		t.Errorf("AllPanic 钩子 panic 应触发 HookPanic")
	}
	if allCalls < 2 {
		t.Errorf("AllPanic 钩子应被触发多次，实际 %d", allCalls)
	}
}

// 测试 Stop 清场时，残留定时器回调 panic 不冒泡且触发 TimerPanic。
// 使用长定时器确保回调在 Stop 时仍残留于 timeline.tasks，由 flushTasks 执行。
func TestStopFlushTasksTimerPanic(t *testing.T) {
	t.Parallel()
	el2 := StartEventLoop(1)

	var timerPanicCalled bool
	el2.OnPanic(TimerPanic, func(r any) {
		timerPanicCalled = true
	})

	// 长定时器：Stop 时回调仍残留于调度队列，flushTasks 清场时执行并触发 TimerPanic。
	// 需短暂等待，确保 scheduler 已从 taskCh 取出并放入调度队列，避免时序竞争。
	el2.SetTimeout(func() {
		panic("timer boom")
	}, 5000)
	time.Sleep(50 * time.Millisecond)

	el2.Stop() // 不应 panic 冒泡
	if !timerPanicCalled {
		t.Errorf("残留定时器回调 panic 应触发 TimerPanic 钩子")
	}
}
