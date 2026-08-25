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
