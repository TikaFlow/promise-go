package promise

import (
	"fmt"
	"time"
)

// 基本使用
func Example_basic() {
	el := StartEventLoop(1)
	defer el.Stop()

	// promise 的使用
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("hello world")
		return nil
	})
	p.Then(func(v any) (any, error) {
		fmt.Println(v.(string))
		return nil, nil
	}, nil)

	// 延时任务
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
	// Output:
	// hello world
	// [A]
	// [B]
	// [C]
}

// 进阶使用
func Example_advance() {
	el := StartEventLoop(1)
	defer el.Stop()

	// 链式调用
	el.Resolve(1).
		Then(func(v any) (any, error) {
			fmt.Println("hello")
			return v, nil
		}, nil).
		Then(func(v any) (any, error) {
			fmt.Println("world")
			return v, nil
		}, nil).
		Catch(func(err error) (any, error) {
			fmt.Println("Nothing happened here.")
			return nil, nil
		}).
		Finally(func() (any, error) {
			fmt.Println("Finally.")
			return nil, nil
		})

	// 批量处理
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
	// Output:
	// hello
	// world
	// hello world1
	// hello world2
	// hello world3
	// Finally.
}
