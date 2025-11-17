package promise_test

import "github.com/TikaFlow/promise-go"

// 嵌套
func NestedExample() {
	promise.SetTimeout(func() {
		println("[A]")
		promise.SetTimeout(func() {
			println("[C]")
		}, 20)
	}, 30)

	promise.SetTimeout(func() {
		println("[B]")
	}, 50)

	<-promise.Done()
}

// 批量处理
func BatchExample() {
	p1 := promise.New(func(resolve, reject func(v any)) (err error) {
		resolve("hello world")
		return
	})
	p2 := promise.New(func(resolve, reject func(v any)) (err error) {
		resolve("hello world")
		return
	})
	p3 := promise.New(func(resolve, reject func(v any)) (err error) {
		resolve("hello world")
		return
	})

	promise.All(p1, p2, p3).Then(func(v any) (any, error) {
		println(v.([]any)[0].(string))
		println(v.([]any)[1].(string))
		println(v.([]any)[2].(string))
		return nil, nil
	}, nil)

	<-promise.Done()
}
