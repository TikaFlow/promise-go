package aplus_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 2.2.7: `then` 必须返回一个 promise：`promise2 = promise1.then(onFulfilled, onRejected)`
func TestAplus2_2_7(t *testing.T) {
	t.Parallel()
	t.Run("is a promise", func(t *testing.T) {
		t.Parallel()
		// adjusted: `promise1.then()` 在 Go 中必须以 `Then(nil, nil)` 显式表达
		p1, _, _ := deferred()
		p2 := p1.Then(nil, nil)
		if p2 == nil {
			t.Errorf("then should return a promise, got nil")
		}
		// *Promise 类型天然拥有 Then 方法，等价于 JS 的 `typeof promise2.then === "function"`
	})

	t.Run("2.2.7.1: If either `onFulfilled` or `onRejected` returns a value `x`, run the Promise Resolution Procedure `[[Resolve]](promise2, x)`", func(t *testing.T) {
		t.Parallel()
		// 占位用例：该子句由独立的 3.3 系列测试覆盖，此处不做断言。
	})

	t.Run("2.2.7.2: If either `onFulfilled` or `onRejected` throws an exception `e`, `promise2` must be rejected with `e` as the reason.", func(t *testing.T) {
		t.Parallel()
		reasons := []struct {
			name string
			make func() any
		}{
			// 每个 reason 在单一用例名内仅创建一次，throw 与断言共用同一实例，
			// 使 assertReason 的 DeepEqual / errors.Is 能得到正确的同一性比较。
			{"`undefined`", func() any { return jsUndefined }},
			{"`null`", func() any { return nil }},
			{"`false`", func() any { return false }},
			{"`0`", func() any { return 0 }},
			{"an error", func() any { return errors.New("an error") }},
			{"an error without a stack", func() any { return errors.New("an error") }},
			{"a date", func() any { return time.Now() }},
			{"an object", func() any { return &struct{ name string }{name: "obj"} }},
			{"an always-pending thenable", func() any { p, _, _ := deferred(); return p }},
			{"a fulfilled promise", func() any { return el.Resolve(dummy) }},
			{"a rejected promise", func() any { return el.Reject(dummy) }},
		}

		for _, r := range reasons {
			reason := r.make()
			t.Run("The reason is "+r.name, func(t *testing.T) {
				// adjusted: throw → return err
				testFulfilled(t, dummy, func(t *testing.T, p *Promise) *Promise {
					p2 := p.Then(func(v any) (any, error) {
						return nil, thrownErr(reason)
					}, nil)
					return p2.Then(nil, func(r error) (any, error) {
						assertReason(t, r, reason)
						return nil, nil
					})
				})
				// adjusted: throw → return err
				testRejected(t, dummy, func(t *testing.T, p *Promise) *Promise {
					p2 := p.Then(nil, func(e error) (any, error) {
						return nil, thrownErr(reason)
					})
					return p2.Then(nil, func(r error) (any, error) {
						assertReason(t, r, reason)
						return nil, nil
					})
				})
			})
		}
	})

	t.Run("2.2.7.3: If `onFulfilled` is not a function and `promise1` is fulfilled, `promise2` must be fulfilled with the same value.", func(t *testing.T) {
		t.Parallel()
		// 官方在此遍历 undefined/null/false/5/object/array 六种非函数值。
		// Go 的 nil 回调即可表达 undefined/null；其余 false/5/object/array 无对应物，跳过。
		testFulfilled(t, sentinel, func(t *testing.T, p *Promise) *Promise {
			return p.Then(nil, func(r error) (any, error) {
				t.Errorf("unexpected reject %v", r)
				return nil, nil
			}).Then(func(v any) (any, error) {
				assertFulfillValue(t, v, sentinel)
				return nil, nil
			}, nil)
		})
	})

	t.Run("2.2.7.3-N/A", func(t *testing.T) {
		t.Parallel()
		skipNA(t, "2.2.7.3-N/A", 15, "非函数处理器中除 nil/undefined/null 外，false/5/object/array 在 Go 无对应物")
	})

	t.Run("2.2.7.4: If `onRejected` is not a function and `promise1` is rejected, `promise2` must be rejected with the same reason.", func(t *testing.T) {
		t.Parallel()
		// 官方在此遍历 undefined/null/false/5/object/array 六种非函数值。
		// Go 的 nil 回调即可表达 undefined/null；其余 false/5/object/array 无对应物，跳过。
		testRejected(t, sentinel, func(t *testing.T, p *Promise) *Promise {
			return p.Then(nil, nil).Then(nil, func(r error) (any, error) {
				assertReason(t, r, sentinel)
				return nil, nil
			})
		})
	})

	t.Run("2.2.7.4-N/A", func(t *testing.T) {
		t.Parallel()
		skipNA(t, "2.2.7.4-N/A", 15, "非函数处理器中除 nil/undefined/null 外，false/5/object/array 在 Go 无对应物")
	})
}
