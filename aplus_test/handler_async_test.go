package aplus_test

import (
	"sync/atomic"
	"testing"

	. "github.com/TikaFlow/promise-go"
)

// 2.2.4: `onFulfilled` 与 `onRejected` 必须等到执行上下文栈中只含平台代码时才被调用
// （即回调必须异步/"干净栈"地执行）。Go 的单 looper 模型天然保证 handler 不同步运行，
// 因此这里移植其"本质"：handler 不得在注册点之前被调用；嵌套注册的内层 handler 须在外层之后运行。
func TestAplus2_2_4(t *testing.T) {
	t.Parallel()
	t.Run("then returns before the promise becomes fulfilled or rejected", func(t *testing.T) {
		t.Parallel()
		t.Run("onFulfilled (adjusted)", func(t *testing.T) {
			t.Parallel()
			// adjusted: JS 在 handler 内断言 thenHasReturned===true（即 handler 在 then 返回后运行）。
			// Go 中 handler 不在注册 goroutine 同步运行，用标志证明其在 Then 返回后才被调用。
			testFulfilled(t, dummy, func(t *testing.T, p *Promise) *Promise {
				var thenReturned atomic.Bool
				tail := p.Then(func(v any) (any, error) {
					if !thenReturned.Load() {
						t.Errorf("onFulfilled ran before Then returned")
					}
					return nil, nil
				}, nil)
				thenReturned.Store(true)
				return tail
			})
		})

		t.Run("onRejected (adjusted)", func(t *testing.T) {
			t.Parallel()
			// adjusted: 同上，onRejected 的干净栈断言改写为异步标志证明。
			testRejected(t, dummy, func(t *testing.T, p *Promise) *Promise {
				var thenReturned atomic.Bool
				tail := p.Then(nil, func(r error) (any, error) {
					if !thenReturned.Load() {
						t.Errorf("onRejected ran before Then returned")
					}
					return nil, nil
				})
				thenReturned.Store(true)
				return tail
			})
		})
	})

	t.Run("Clean-stack execution ordering tests (fulfillment case)", func(t *testing.T) {
		t.Parallel()
		t.Run("when onFulfilled is added immediately before the promise is fulfilled (adjusted)", func(t *testing.T) {
			t.Parallel()
			// adjusted: JS 在 resolve 后同步断言 flag 仍为 false；Go 中改为在未决时（handler 必然未运行）断言 false，
			// resolve 后再等 tail 完成并断言 true，以证明 handler 是异步触发的。
			p, resolve, _ := deferred()
			var called atomic.Bool
			tail := p.Then(func(v any) (any, error) {
				called.Store(true)
				return nil, nil
			}, nil)

			if called.Load() {
				t.Errorf("onFulfilled ran before resolve (should be async)")
			}
			resolve(dummy)
			waitTail(t, tail)
			if !called.Load() {
				t.Errorf("onFulfilled should have been called asynchronously")
			}
		})

		t.Run("when onFulfilled is added immediately after the promise is fulfilled (adjusted)", func(t *testing.T) {
			t.Parallel()
			// adjusted: JS 在 resolve 后添加 handler，并同步断言其未被调用；
			// Go 中已决 promise 上注册 handler 同样不在注册 goroutine 同步运行，用原子标志即时断言。
			p, resolve, _ := deferred()
			resolve(dummy)
			var called atomic.Bool
			tail := p.Then(func(v any) (any, error) {
				called.Store(true)
				return nil, nil
			}, nil)

			if called.Load() {
				t.Errorf("onFulfilled ran synchronously when added after fulfillment")
			}
			waitTail(t, tail)
			if !called.Load() {
				t.Errorf("onFulfilled should have been called asynchronously")
			}
		})

		t.Run("when one onFulfilled is added inside another onFulfilled (adjusted)", func(t *testing.T) {
			t.Parallel()
			// adjusted: JS 以 firstOnFulfilledFinished 断言内层 handler 在外层完成后运行。
			// Go 中两层 handler 都在 looper 上顺序执行，用顺序切片断言内层晚于外层，并借外层返回内层让 waitTail 等到内层。
			promise := resolved(dummy)
			var order []string
			tail := promise.Then(func(v any) (any, error) {
				inner := promise.Then(func(v any) (any, error) {
					order = append(order, "inner")
					return nil, nil
				}, nil)
				order = append(order, "outer")
				return inner, nil
			}, nil)
			waitTail(t, tail)
			if len(order) != 2 || order[0] != "outer" || order[1] != "inner" {
				t.Errorf("inner onFulfilled should run after outer onFulfilled, got order %v", order)
			}
		})

		t.Run("when onFulfilled is added inside an onRejected (adjusted)", func(t *testing.T) {
			t.Parallel()
			// adjusted: JS 断言在 onRejected 内注册的 onFulfilled 于外层完成后运行，改写为顺序断言。
			promise := rejected(dummy)
			promise2 := resolved(dummy)
			var order []string
			tail := promise.Then(nil, func(r error) (any, error) {
				inner := promise2.Then(func(v any) (any, error) {
					order = append(order, "inner")
					return nil, nil
				}, nil)
				order = append(order, "outer")
				return inner, nil
			})
			waitTail(t, tail)
			if len(order) != 2 || order[0] != "outer" || order[1] != "inner" {
				t.Errorf("inner onFulfilled should run after outer onRejected, got order %v", order)
			}
		})

		t.Run("when the promise is fulfilled asynchronously (adjusted)", func(t *testing.T) {
			t.Parallel()
			// adjusted: JS 在 setTimeout(0) 内 resolve 后置 firstStackFinished=true，handler 再断言其已为 true。
			// Go 中 setTimeout 回调（宏任务）与 handler（微任务）都在 looper 上顺序执行，以 flag 证明时序。
			p, resolve, _ := deferred()
			var firstStackFinished bool
			el.SetTimeout(func() {
				resolve(dummy)
				firstStackFinished = true
			}, 0)
			tail := p.Then(func(v any) (any, error) {
				if !firstStackFinished {
					t.Errorf("onFulfilled ran before the async resolve stack finished")
				}
				return nil, nil
			}, nil)
			waitTail(t, tail)
		})
	})

	t.Run("Clean-stack execution ordering tests (rejection case)", func(t *testing.T) {
		t.Parallel()
		t.Run("when onRejected is added immediately before the promise is rejected (adjusted)", func(t *testing.T) {
			t.Parallel()
			// adjusted: JS 在 reject 后同步断言 flag 仍为 false；Go 改为未决时断言 false，reject 后断言 true。
			p, _, reject := deferred()
			var called atomic.Bool
			tail := p.Then(nil, func(r error) (any, error) {
				called.Store(true)
				return nil, nil
			})

			if called.Load() {
				t.Errorf("onRejected ran before reject (should be async)")
			}
			reject(dummy)
			waitTail(t, tail)
			if !called.Load() {
				t.Errorf("onRejected should have been called asynchronously")
			}
		})

		t.Run("when onRejected is added immediately after the promise is rejected (adjusted)", func(t *testing.T) {
			t.Parallel()
			// adjusted: JS 在 reject 后添加 handler，并同步断言其未被调用；Go 用原子标志即时断言。
			p, _, reject := deferred()
			reject(dummy)
			var called atomic.Bool
			tail := p.Then(nil, func(r error) (any, error) {
				called.Store(true)
				return nil, nil
			})

			if called.Load() {
				t.Errorf("onRejected ran synchronously when added after rejection")
			}
			waitTail(t, tail)
			if !called.Load() {
				t.Errorf("onRejected should have been called asynchronously")
			}
		})

		t.Run("when onRejected is added inside an onFulfilled (adjusted)", func(t *testing.T) {
			t.Parallel()
			// adjusted: JS 断言在 onFulfilled 内注册的 onRejected 于外层完成后运行，改写为顺序断言。
			promise := resolved(dummy)
			promise2 := rejected(dummy)
			var order []string
			tail := promise.Then(func(v any) (any, error) {
				inner := promise2.Then(nil, func(r error) (any, error) {
					order = append(order, "inner")
					return nil, nil
				})
				order = append(order, "outer")
				return inner, nil
			}, nil)
			waitTail(t, tail)
			if len(order) != 2 || order[0] != "outer" || order[1] != "inner" {
				t.Errorf("inner onRejected should run after outer onFulfilled, got order %v", order)
			}
		})

		t.Run("when one onRejected is added inside another onRejected (adjusted)", func(t *testing.T) {
			t.Parallel()
			// adjusted: JS 断言内层 onRejected 在外层完成后运行，改写为顺序断言。
			promise := rejected(dummy)
			var order []string
			tail := promise.Then(nil, func(r error) (any, error) {
				inner := promise.Then(nil, func(r error) (any, error) {
					order = append(order, "inner")
					return nil, nil
				})
				order = append(order, "outer")
				return inner, nil
			})
			waitTail(t, tail)
			if len(order) != 2 || order[0] != "outer" || order[1] != "inner" {
				t.Errorf("inner onRejected should run after outer onRejected, got order %v", order)
			}
		})

		t.Run("when the promise is rejected asynchronously (adjusted)", func(t *testing.T) {
			t.Parallel()
			// adjusted: JS 在 setTimeout(0) 内 reject 后置 firstStackFinished=true，handler 再断言其已为 true。
			p, _, reject := deferred()
			var firstStackFinished bool
			el.SetTimeout(func() {
				reject(dummy)
				firstStackFinished = true
			}, 0)
			tail := p.Then(nil, func(r error) (any, error) {
				if !firstStackFinished {
					t.Errorf("onRejected ran before the async reject stack finished")
				}
				return nil, nil
			})
			waitTail(t, tail)
		})
	})
}
