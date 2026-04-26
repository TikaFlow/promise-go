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
	el2 := StartEventLoop(1)
	defer el2.Stop()

	result := ""
	_ = el2.On(OnCreated, func(p *Promise) {
		result += "created-"
	})
	_ = el2.On(OnSettled, func(p *Promise) {
		result += "settled-"
	})
	_ = el2.On(OnFulfilled, func(p *Promise) {
		result += "fulfilled-"
	})
	_ = el2.On(OnRejected, func(p *Promise) {
		result += "rejected-"
	})
	_ = el2.On(OnChained, func(p *Promise) {
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
	time.Sleep(100 * time.Millisecond)
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
	time.Sleep(100 * time.Millisecond)
	expected += "settled-fulfilled-settled-fulfilled-"
	if result != expected {
		t.Errorf("result should be '%s', but got '%s'", expected, result)
	}
}

// 测试Off方法 - 成功解绑
func TestOffSuccess(t *testing.T) {
	el2 := StartEventLoop(1)
	defer el2.Stop()

	called := false
	key := el2.On(OnCreated, func(p *Promise) {
		called = true
	})

	result := el2.Off(OnCreated, key)
	if !result {
		t.Errorf("Expected Off to return true")
	}

	el2.NewPromise(func(resolve, reject func(v any)) error {
		resolve("test")
		return nil
	})

	time.Sleep(100 * time.Millisecond)
	if called {
		t.Errorf("Expected hook to not be called after Off")
	}
}

// 测试Off - 不匹配的key/hookType
func TestOffMismatch(t *testing.T) {
	el2 := StartEventLoop(1)
	defer el2.Stop()

	key := el.On(OnCreated, func(p *Promise) {})
	result := el.Off(OnRejected, key)
	if result {
		t.Errorf("Expected Off to return false for mismatched event type")
	}

	key = el.On(OnChained, func(p *Promise) {})
	result = el.Off(OnChained, "this-is-a-key")
	if result {
		t.Errorf("Expected Off to return false for mismatched key")
	}
}
