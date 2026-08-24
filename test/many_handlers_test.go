package promise_test

import (
	"testing"
	"time"
)

// 回归：对未决 promise 注册超过旧有 128 缓冲上限的 then，不应死锁，且应全部、按序执行（规范 2.2.6）。
func TestManyThenOnPendingPromise(t *testing.T) {
	t.Parallel()
	const n = 130 // 略大于旧有缓冲上限 128（修复后第 129 次注册不再阻塞）
	p, resolve, _ := el.PromiseWithResolvers()

	// 用带缓冲 channel 收集回调执行的索引，避免跨 goroutine 数据竞争。
	ch := make(chan int, n)
	for i := range n {
		idx := i
		p.Then(func(v any) (any, error) {
			ch <- idx
			return nil, nil
		}, nil)
	}
	resolve("done")

	deadline := time.After(3 * time.Second)
	for i := range n {
		select {
		case got := <-ch:
			if got != i {
				t.Fatalf("order mismatch at %d: got %d", i, got)
			}
		case <-deadline:
			t.Fatalf("only %d/%d onFulfilled ran", i, n)
		}
	}
}
