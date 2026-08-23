package aplus_test

import (
	"reflect"
	"sync/atomic"
	"testing"

	. "github.com/TikaFlow/promise-go"
)

// TestAplus2_2_6 对应官方套件 2.2.6：`then` 可在同一个 promise 上多次调用。
// 当 promise 被 fulfilled 时，所有 onFulfilled 按调用 then 的顺序执行；
// 当 promise 被 rejected 时，所有 onRejected 按调用 then 的顺序执行。
// 其中某个回调抛错不应影响其余兄弟回调；各分支链可自行产生不同的 fulfillment 值。
func TestAplus2_2_6(t *testing.T) {
	//
	// 2.2.6.1 当 promise 被 fulfilled 时，所有 onFulfilled 按 then 注册顺序执行
	//
	t.Run("2.2.6.1", func(t *testing.T) {
		t.Run("multiple boring fulfillment handlers", func(t *testing.T) {
			testFulfilled(t, sentinel, func(t *testing.T, p *Promise) *Promise {
				var order []string
				// adjusted: sinon.stub().returns(other) → 记录 id 并返回 other 的处理器
				handler := func(id string) func(any) (any, error) {
					return func(v any) (any, error) {
						order = append(order, id)
						assertFulfillValue(t, v, sentinel)
						return other, nil
					}
				}
				// adjusted: sinon.spy() → 仅记录是否被调用的处理器
				var spyCalled atomic.Bool
				spy := func(error) (any, error) {
					spyCalled.Store(true)
					return nil, nil
				}

				p.Then(handler("handler1"), spy)
				p.Then(handler("handler2"), spy)
				p.Then(handler("handler3"), spy)

				return p.Then(func(v any) (any, error) {
					assertFulfillValue(t, v, sentinel)
					// adjusted: sinon.assert.calledWith(handler_i, sentinel) → 三者均被调用（且按序）
					want := []string{"handler1", "handler2", "handler3"}
					if !reflect.DeepEqual(order, want) {
						t.Errorf("onFulfilled order = %v, want %v", order, want)
					}
					if spyCalled.Load() {
						t.Errorf("onRejected spy should not be called")
					}
					return nil, nil
				}, nil)
			})
		})

		t.Run("multiple fulfillment handlers, one of which throws", func(t *testing.T) {
			testFulfilled(t, sentinel, func(t *testing.T, p *Promise) *Promise {
				var order []string
				handler := func(id string) func(any) (any, error) {
					return func(v any) (any, error) {
						order = append(order, id)
						assertFulfillValue(t, v, sentinel)
						return other, nil
					}
				}
				// adjusted: sinon.stub().throws(other) → 返回 err 的处理器（其派生 promise 被拒绝，不影响兄弟 handler）
				thrower := func(v any) (any, error) {
					order = append(order, "handler2")
					assertFulfillValue(t, v, sentinel)
					return nil, thrownErr(other)
				}
				var spyCalled atomic.Bool
				spy := func(error) (any, error) {
					spyCalled.Store(true)
					return nil, nil
				}

				p.Then(handler("handler1"), spy)
				p.Then(thrower, spy)
				p.Then(handler("handler3"), spy)

				return p.Then(func(v any) (any, error) {
					assertFulfillValue(t, v, sentinel)
					want := []string{"handler1", "handler2", "handler3"}
					if !reflect.DeepEqual(order, want) {
						t.Errorf("onFulfilled order = %v, want %v", order, want)
					}
					if spyCalled.Load() {
						t.Errorf("onRejected spy should not be called")
					}
					return nil, nil
				}, nil)
			})
		})

		t.Run("results in multiple branching chains with their own fulfillment values", func(t *testing.T) {
			testFulfilled(t, dummy, func(t *testing.T, p *Promise) *Promise {
				// adjusted: callbackAggregator(3, done) → 计数凑满 3 后 resolve 组合 promise
				combiner, resolveCombiner, _ := deferred()
				var settled int32
				semiDone := func() {
					if atomic.AddInt32(&settled, 1) == 3 {
						resolveCombiner(nil)
					}
				}

				p.Then(func(any) (any, error) { return sentinel, nil }, nil).
					Then(func(v any) (any, error) {
						assertFulfillValue(t, v, sentinel)
						return nil, nil
					}, nil).
					Then(func(any) (any, error) { semiDone(); return nil, nil },
						func(error) (any, error) { semiDone(); return nil, nil })

				p.Then(func(any) (any, error) { return nil, thrownErr(sentinel2) }, nil).
					Then(nil, func(r error) (any, error) {
						assertReason(t, r, sentinel2)
						return nil, nil
					}).
					Then(func(any) (any, error) { semiDone(); return nil, nil },
						func(error) (any, error) { semiDone(); return nil, nil })

				p.Then(func(any) (any, error) { return sentinel3, nil }, nil).
					Then(func(v any) (any, error) {
						assertFulfillValue(t, v, sentinel3)
						return nil, nil
					}, nil).
					Then(func(any) (any, error) { semiDone(); return nil, nil },
						func(error) (any, error) { semiDone(); return nil, nil })

				return combiner
			})
		})

		t.Run("'onFulfilled' handlers are called in the original order", func(t *testing.T) {
			testFulfilled(t, dummy, func(t *testing.T, p *Promise) *Promise {
				var order []string
				// adjusted: sinon.spy(handlerN) → 记录 id 的处理器
				handler := func(id string) func(any) (any, error) {
					return func(any) (any, error) {
						order = append(order, id)
						return nil, nil
					}
				}

				p.Then(handler("handler1"), nil)
				p.Then(handler("handler2"), nil)
				p.Then(handler("handler3"), nil)

				// adjusted: sinon.assert.callOrder(handler1, handler2, handler3) → 顺序切片断言
				return p.Then(func(any) (any, error) {
					want := []string{"handler1", "handler2", "handler3"}
					if !reflect.DeepEqual(order, want) {
						t.Errorf("onFulfilled order = %v, want %v", order, want)
					}
					return nil, nil
				}, nil)
			})
		})

		t.Run("even when one handler is added inside another handler", func(t *testing.T) {
			testFulfilled(t, dummy, func(t *testing.T, p *Promise) *Promise {
				var order []string
				handler := func(id string) func(any) (any, error) {
					return func(any) (any, error) {
						order = append(order, id)
						return nil, nil
					}
				}
				combiner, resolveCombiner, _ := deferred()

				p.Then(func(any) (any, error) {
					order = append(order, "handler1")
					// adjusted: 在 handler1 内部再注册 handler3
					p.Then(handler("handler3"), nil)
					return nil, nil
				}, nil)
				p.Then(handler("handler2"), nil)
				p.Then(func(any) (any, error) {
					// adjusted: JS 用 setTimeout 留出微任务排空时间 → 用 SetTimeout 在宏任务中断言
					el.SetTimeout(func() {
						want := []string{"handler1", "handler2", "handler3"}
						if !reflect.DeepEqual(order, want) {
							t.Errorf("onFulfilled order = %v, want %v", order, want)
						}
						resolveCombiner(nil)
					}, 15)
					return nil, nil
				}, nil)

				return combiner
			})
		})
	})

	//
	// 2.2.6.2 当 promise 被 rejected 时，所有 onRejected 按 then 注册顺序执行
	//
	t.Run("2.2.6.2", func(t *testing.T) {
		t.Run("multiple boring rejection handlers", func(t *testing.T) {
			testRejected(t, sentinel, func(t *testing.T, p *Promise) *Promise {
				var order []string
				// adjusted: sinon.stub().returns(other) → 记录 id 并返回 other 的处理器
				handler := func(id string) func(error) (any, error) {
					return func(r error) (any, error) {
						order = append(order, id)
						assertReason(t, r, sentinel)
						return other, nil
					}
				}
				var spyCalled atomic.Bool
				spy := func(any) (any, error) {
					spyCalled.Store(true)
					return nil, nil
				}

				p.Then(spy, handler("handler1"))
				p.Then(spy, handler("handler2"))
				p.Then(spy, handler("handler3"))

				return p.Then(nil, func(r error) (any, error) {
					assertReason(t, r, sentinel)
					want := []string{"handler1", "handler2", "handler3"}
					if !reflect.DeepEqual(order, want) {
						t.Errorf("onRejected order = %v, want %v", order, want)
					}
					if spyCalled.Load() {
						t.Errorf("onFulfilled spy should not be called")
					}
					return nil, nil
				})
			})
		})

		t.Run("multiple rejection handlers, one of which throws", func(t *testing.T) {
			testRejected(t, sentinel, func(t *testing.T, p *Promise) *Promise {
				var order []string
				handler := func(id string) func(error) (any, error) {
					return func(r error) (any, error) {
						order = append(order, id)
						assertReason(t, r, sentinel)
						return other, nil
					}
				}
				// adjusted: sinon.stub().throws(other) → 返回 err 的处理器（其派生 promise 被拒绝，不影响兄弟 handler）
				thrower := func(r error) (any, error) {
					order = append(order, "handler2")
					assertReason(t, r, sentinel)
					return nil, thrownErr(other)
				}
				var spyCalled atomic.Bool
				spy := func(any) (any, error) {
					spyCalled.Store(true)
					return nil, nil
				}

				p.Then(spy, handler("handler1"))
				p.Then(spy, thrower)
				p.Then(spy, handler("handler3"))

				return p.Then(nil, func(r error) (any, error) {
					assertReason(t, r, sentinel)
					want := []string{"handler1", "handler2", "handler3"}
					if !reflect.DeepEqual(order, want) {
						t.Errorf("onRejected order = %v, want %v", order, want)
					}
					if spyCalled.Load() {
						t.Errorf("onFulfilled spy should not be called")
					}
					return nil, nil
				})
			})
		})

		t.Run("results in multiple branching chains with their own fulfillment values", func(t *testing.T) {
			testRejected(t, sentinel, func(t *testing.T, p *Promise) *Promise {
				// adjusted: callbackAggregator(3, done) → 计数凑满 3 后 resolve 组合 promise
				combiner, resolveCombiner, _ := deferred()
				var settled int32
				semiDone := func() {
					if atomic.AddInt32(&settled, 1) == 3 {
						resolveCombiner(nil)
					}
				}

				p.Then(nil, func(error) (any, error) { return sentinel, nil }).
					Then(func(v any) (any, error) {
						assertFulfillValue(t, v, sentinel)
						return nil, nil
					}, nil).
					Then(func(any) (any, error) { semiDone(); return nil, nil },
						func(error) (any, error) { semiDone(); return nil, nil })

				p.Then(nil, func(error) (any, error) { return nil, thrownErr(sentinel2) }).
					Then(nil, func(r error) (any, error) {
						assertReason(t, r, sentinel2)
						return nil, nil
					}).
					Then(func(any) (any, error) { semiDone(); return nil, nil },
						func(error) (any, error) { semiDone(); return nil, nil })

				p.Then(nil, func(error) (any, error) { return sentinel3, nil }).
					Then(func(v any) (any, error) {
						assertFulfillValue(t, v, sentinel3)
						return nil, nil
					}, nil).
					Then(func(any) (any, error) { semiDone(); return nil, nil },
						func(error) (any, error) { semiDone(); return nil, nil })

				return combiner
			})
		})

		t.Run("'onRejected' handlers are called in the original order", func(t *testing.T) {
			testRejected(t, dummy, func(t *testing.T, p *Promise) *Promise {
				var order []string
				// adjusted: sinon.spy(handlerN) → 记录 id 的处理器
				handler := func(id string) func(error) (any, error) {
					return func(error) (any, error) {
						order = append(order, id)
						return nil, nil
					}
				}

				p.Then(nil, handler("handler1"))
				p.Then(nil, handler("handler2"))
				p.Then(nil, handler("handler3"))

				// adjusted: sinon.assert.callOrder(handler1, handler2, handler3) → 顺序切片断言
				return p.Then(nil, func(error) (any, error) {
					want := []string{"handler1", "handler2", "handler3"}
					if !reflect.DeepEqual(order, want) {
						t.Errorf("onRejected order = %v, want %v", order, want)
					}
					return nil, nil
				})
			})
		})

		t.Run("even when one handler is added inside another handler", func(t *testing.T) {
			testRejected(t, dummy, func(t *testing.T, p *Promise) *Promise {
				var order []string
				handler := func(id string) func(error) (any, error) {
					return func(error) (any, error) {
						order = append(order, id)
						return nil, nil
					}
				}
				combiner, resolveCombiner, _ := deferred()

				p.Then(nil, func(error) (any, error) {
					order = append(order, "handler1")
					// adjusted: 在 handler1 内部再注册 handler3
					p.Then(nil, handler("handler3"))
					return nil, nil
				})
				p.Then(nil, handler("handler2"))
				p.Then(nil, func(error) (any, error) {
					// adjusted: JS 用 setTimeout 留出微任务排空时间 → 用 SetTimeout 在宏任务中断言
					el.SetTimeout(func() {
						want := []string{"handler1", "handler2", "handler3"}
						if !reflect.DeepEqual(order, want) {
							t.Errorf("onRejected order = %v, want %v", order, want)
						}
						resolveCombiner(nil)
					}, 15)
					return nil, nil
				})

				return combiner
			})
		})
	})
}
