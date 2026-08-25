package aplus_test

import (
	"testing"

	. "github.com/TikaFlow/promise-go"
)

// 2.3.4: 若 x 不是对象或函数，则以 x 的值 fulfill promise。
// 从 fulfilled 与 rejected 处理器返回普通值，断言下游收到该值。
func TestAplus2_3_4(t *testing.T) {
	t.Parallel()
	values := []struct {
		value any
		name  string
	}{
		{jsUndefined, "the value is undefined"},
		{nil, "the value is null"},
		{false, "the value is false"},
		{true, "the value is true"},
		{0, "the value is 0"},
	}

	for _, v := range values {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			testFulfilled(t, dummy, func(t *testing.T, p *Promise) *Promise {
				p2 := p.Then(func(any) (any, error) { return v.value, nil }, nil)
				return expectFulfilled(t, p2, v.value)
			})
			testRejected(t, dummy, func(t *testing.T, p *Promise) *Promise {
				p2 := p.Then(nil, func(error) (any, error) { return v.value, nil })
				return expectFulfilled(t, p2, v.value)
			})
		})
	}

	skipNA(t, "2.3.4 boxed prototype with then", 12,
		"Go 无原型链，无法在 Boolean.prototype / Number.prototype 上挂 then（true/1 各 6 例）")
}
