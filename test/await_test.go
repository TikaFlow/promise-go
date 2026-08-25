package promise_test

import (
	"sync/atomic"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 回归：Await 在基 promise 先于超时已决时，应清理超时定时器，避免 wait promise
// 在超时点被无谓地拒绝（触发 PromiseRejected 等副作用）。
func TestAwaitClearsTimeoutWhenBaseSettlesFirst(t *testing.T) {
	t.Parallel()
	// 使用独立事件循环，避免钩子影响其他测试（与 hook_test 同约定）。
	el2 := StartEventLoop(1)
	defer el2.Stop()

	var rejected atomic.Bool
	el2.OnPromise(PromiseRejected, func(*Promise) {
		rejected.Store(true)
	})

	p, resolve, _ := el2.PromiseWithResolvers()
	el2.SetTimeout(func() { resolve("ok") }, 30)
	v, err := el2.Await(p, 500)
	if err != nil || v != "ok" {
		t.Fatalf("expected ('ok', nil), got (v=%v, err=%v)", v, err)
	}

	// 等待超过超时点：若定时器未被清理，wait promise 会被拒绝并触发 PromiseRejected。
	time.Sleep(time.Second)
	if rejected.Load() {
		t.Fatal("timeout timer was not cleared: wait promise got rejected after base settled")
	}
}
