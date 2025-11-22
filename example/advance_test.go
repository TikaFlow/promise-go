package promise_test

import (
	"fmt"
	"time"

	"github.com/TikaFlow/promise-go"
)

// 嵌套
func ExampleSetTimeout_nested() {
	promise.SetTimeout(func() {
		fmt.Println("[A]")
		promise.SetTimeout(func() {
			fmt.Println("[C]")
		}, 20)
	}, 30)

	promise.SetTimeout(func() {
		fmt.Println("[B]")
	}, 50)

	time.Sleep(time.Millisecond * 100)

	// Output:
	// [A]
	// [B]
	// [C]
}

// 批量处理
func ExampleAll() {
	p1 := promise.New(func(resolve, reject func(v any)) (err any) {
		resolve("hello world1")
		return
	})
	p2 := promise.New(func(resolve, reject func(v any)) (err any) {
		resolve("hello world2")
		return
	})
	p3 := promise.New(func(resolve, reject func(v any)) (err any) {
		resolve("hello world3")
		return
	})

	promise.All(p1, p2, p3).Then(func(v any) (any, any) {
		fmt.Println(v.([]any)[0].(string))
		fmt.Println(v.([]any)[1].(string))
		fmt.Println(v.([]any)[2].(string))
		return nil, nil
	}, nil)

	time.Sleep(time.Millisecond * 50)

	// Output:
	// hello world1
	// hello world2
	// hello world3
}
