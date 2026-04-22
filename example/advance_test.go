package promise_test

import (
	"fmt"
	"time"

	"github.com/TikaFlow/promise-go"
)

// 嵌套
func Example_setTimeout_nested() {
	el := promise.StartClassicEventLoop()
	el.SetTimeout(func() {
		fmt.Println("[A]")
		el.SetTimeout(func() {
			fmt.Println("[C]")
		}, 20)
	}, 30)

	el.SetTimeout(func() {
		fmt.Println("[B]")
	}, 50)

	time.Sleep(time.Millisecond * 100)

	_ = el.Close()
	// Output:
	// [A]
	// [B]
	// [C]
}

// 批量处理
func Example_all() {
	el := promise.StartClassicEventLoop()
	p1 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("hello world1")
		return nil
	})
	p2 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("hello world2")
		return nil
	})
	p3 := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("hello world3")
		return nil
	})

	el.All(p1, p2, p3).Then(func(v any) (any, error) {
		fmt.Println(v.([]any)[0].(string))
		fmt.Println(v.([]any)[1].(string))
		fmt.Println(v.([]any)[2].(string))
		return nil, nil
	}, nil)

	time.Sleep(time.Millisecond * 50)

	_ = el.Close()
	// Output:
	// hello world1
	// hello world2
	// hello world3
}
