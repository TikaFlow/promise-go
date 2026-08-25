package aplus_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 本文件实现官方 promises-aplus-tests 套件所需的 adapter 三件套、helper 展开函数与断言工具。

// ---- 哨兵值 ----
// 对应 JS 套件中的 dummy / sentinel / sentinel2 / sentinel3 / other / undefined。
// Go 中“对象严格相等”以指针同一性表示，因此采用指针哨兵（可比较）。
var (
	dummy     = &struct{ name string }{name: "dummy"}
	sentinel  = &struct{ name string }{name: "sentinel"}
	sentinel2 = &struct{ name string }{name: "sentinel2"}
	sentinel3 = &struct{ name string }{name: "sentinel3"}
	other     = &struct{ name string }{name: "other"}
	// Go 无 undefined；null 映射为 nil。jsUndefined 用于需要区分 undefined 语义的场合。
	jsUndefined = &struct{ name string }{name: "undefined"}
)

// waitTimeout 等待 promise 已决的超时上限，远大于官方 mocha 默认 200ms，且足够抗负载。
const waitTimeout = 1500 * time.Millisecond

// ---- adapter 三件套（JS: { resolved, rejected, deferred }）----
func resolved(v any) *Promise { return el.Resolve(v) }
func rejected(r any) *Promise { return el.Reject(r) }

func deferred() (*Promise, func(any), func(any)) {
	return el.PromiseWithResolvers()
}

// ---- 等待工具 ----
// waitDone 阻塞至 p 已决（Done() 通道关闭），或超时报错。p 须非 nil。
func waitDone(t *testing.T, p *Promise, d time.Duration) {
	t.Helper()
	select {
	case <-p.Done():
	case <-time.After(d):
		t.Errorf("timeout waiting for promise to settle (state=%s)", p.State())
	}
}

// waitTail 等待 test 回调返回的“尾部 promise”。返回 nil 表示该用例自同步
// （如“仍 pending”场景），无需统一等待。
func waitTail(t *testing.T, tail *Promise) {
	t.Helper()
	if tail == nil {
		return
	}
	waitDone(t, tail, waitTimeout)
}

// ---- 断言便利 ----
// assertFulfillValue 断言 fulfillment 值与 want 严格相等。
// 指针用引用相等（形同 JS strictEqual），基本值用值相等。
func assertFulfillValue(t *testing.T, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("expected fulfillment value %#v, got %#v", want, got)
	}
}

// assertErrorType 断言 err 是类型 T（如 *TypeError）。
func assertErrorType[T error](t *testing.T, err error, want T) {
	t.Helper()
	var target T
	if !errors.As(err, &target) {
		t.Errorf("expected error of type %T, got %v", want, err)
	}
}

// thrownErr 把回调“抛出”的任意值规范化为拒绝理由：error 原样返回，其余包装为 *UnexpectedError。
func thrownErr(rv any) error {
	if e, ok := rv.(error); ok {
		return e
	}
	return NewUnexpectedError(rv)
}

// assertReason 断言拒绝理由与 want 相符。
//   - want 为 error 时按 errors.Is 判断（同一实例）。
//   - 否则因实现会把非 error 理由包装为 *UnexpectedError，且其 reason 字段未导出，
//     故用 errors.As 确认类型后，以 reflect.DeepEqual 穿透比较内层 reason。
//     由于各处复用同一个 want 实例，DeepEqual 会走指针/值相等捷径，不会深递归。
func assertReason(t *testing.T, got error, want any) {
	t.Helper()
	if e, ok := want.(error); ok {
		if !errors.Is(got, e) {
			t.Errorf("expected reason %v, got %v", e, got)
		}
		return
	}

	var ue *UnexpectedError
	if !errors.As(got, &ue) {
		t.Errorf("expected UnexpectedError wrapping %#v, got %v", want, got)
		return
	}
	if !reflect.DeepEqual(ue, NewUnexpectedError(want)) {
		t.Errorf("expected UnexpectedError {%v}, got %v", want, got)
	}
}

// expectFulfilled 断言 p 以值 want 完成，返回该断言链的尾部 promise。
func expectFulfilled(t *testing.T, p *Promise, want any) *Promise {
	return p.Then(func(v any) (any, error) {
		assertFulfillValue(t, v, want)
		return nil, nil
	}, func(r error) (any, error) {
		t.Errorf("expected fulfilled %#v, got rejected %v", want, r)
		return nil, nil
	})
}

// expectRejected 断言 p 以理由 wantReason 拒绝，返回该断言链的尾部 promise。
func expectRejected(t *testing.T, p *Promise, wantReason any) *Promise {
	return p.Then(func(v any) (any, error) {
		t.Errorf("expected rejected, got fulfilled %#v", v)
		return nil, nil
	}, func(r error) (any, error) {
		assertReason(t, r, wantReason)
		return nil, nil
	})
}

// ---- helper 三剑客（对应 JS helpers/testThreeCases.js）----

// testFunc 是每条用例的回调：接收被测 promise，返回可等待的“尾部 promise”。
// 返回 nil 表示该用例自同步（如“应保持 pending”的断言由回调内部 sleep 完成）。
type testFunc func(t *testing.T, p *Promise) *Promise

// testFulfilled 用 value 作为 fulfillment 值，按 already / immediately / eventually 三重展开。
func testFulfilled(t *testing.T, value any, test testFunc) {
	t.Helper()
	t.Run("already-fulfilled", func(t *testing.T) {
		waitTail(t, test(t, resolved(value)))
	})
	t.Run("immediately-fulfilled", func(t *testing.T) {
		p, resolve, _ := deferred()
		tail := test(t, p)
		resolve(value)
		waitTail(t, tail)
	})
	t.Run("eventually-fulfilled", func(t *testing.T) {
		p, resolve, _ := deferred()
		tail := test(t, p)
		el.SetTimeout(func() { resolve(value) }, 50)
		waitTail(t, tail)
	})
}

// testRejected 用 reason 作为拒绝理由，按 already / immediately / eventually 三重展开。
func testRejected(t *testing.T, reason any, test testFunc) {
	t.Helper()
	t.Run("already-rejected", func(t *testing.T) {
		waitTail(t, test(t, rejected(reason)))
	})
	t.Run("immediately-rejected", func(t *testing.T) {
		p, _, reject := deferred()
		tail := test(t, p)
		reject(reason)
		waitTail(t, tail)
	})
	t.Run("eventually-rejected", func(t *testing.T) {
		p, _, reject := deferred()
		tail := test(t, p)
		el.SetTimeout(func() { reject(reason) }, 50)
		waitTail(t, tail)
	})
}

// testPromiseResolution 经“从 fulfilled 返回”与“从 rejected 返回”两条分支调用 xFactory，
// 对应 JS helpers 中的 testPromiseResolution。
func testPromiseResolution(t *testing.T, xFactory func() any, test testFunc) {
	t.Helper()
	t.Run("via return from a fulfilled promise", func(t *testing.T) {
		p := el.Resolve(dummy).Then(func(any) (any, error) { return xFactory(), nil }, nil)
		waitTail(t, test(t, p))
	})
	t.Run("via return from a rejected promise", func(t *testing.T) {
		p := el.Reject(dummy).Then(nil, func(error) (any, error) { return xFactory(), nil })
		waitTail(t, test(t, p))
	})
}

// skipNA 用一个带原因的 SKIP 子测试归并一段“不适用/故意不支持”的官方用例，供 -v 查看。
func skipNA(t *testing.T, name string, n int, reason string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Skipf("N/A: %s (官方 %d 个用例移植时跳过)", reason, n)
	})
}
