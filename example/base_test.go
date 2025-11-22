package promise_test

import (
	"fmt"
	"time"

	"github.com/TikaFlow/promise-go"
)

// 基础用法
func Example_base() {
	p := promise.New(func(resolve, reject func(v any)) (err any) {
		resolve("hello world")
		return
	})

	p.Then(func(v any) (any, any) {
		fmt.Println(v.(string))
		return nil, nil
	}, nil)

	time.Sleep(time.Millisecond * 50)

	// Output:
	// hello world
}

// 链式调用
func Example_chain() {
	p := promise.New(func(resolve, reject func(v any)) (err any) {
		resolve("hello world")
		return
	})

	p.Then(func(v any) (any, any) {
		fmt.Println(v.(string))
		return v, nil
	}, nil).Then(func(v any) (any, any) {
		fmt.Println(v.(string))
		return v, nil
	}, nil).Catch(func(err any) (any, any) {
		fmt.Println("Nothing happend here.")
		return nil, nil
	}).Finally(func() (any, any) {
		fmt.Println("Finally.")
		return nil, nil
	})

	time.Sleep(time.Millisecond * 50)

	// Output:
	// hello world
	// hello world
	// Finally.
}

// 宏任务 - 见 [Example_delay]

// 微队列
func ExampleQueueMicrotask() {
	promise.QueueMicrotask(func() {
		fmt.Println("Microtask 1")
	})
	promise.QueueMicrotask(func() {
		fmt.Println("Microtask 2")
	})

	time.Sleep(time.Millisecond * 50)

	// Output:
	// Microtask 1
	// Microtask 2
}

// 延迟执行
func Example_delay() {
	id1 := promise.SetTimeout(func() {
		fmt.Println("Timeout 1")
	}, 100)

	var id2 int
	id2 = promise.SetInterval(func() {
		promise.ClearTimeout(id1)
		promise.ClearInterval(id2)
		fmt.Println("Interval 1")
	}, 50)

	time.Sleep(time.Millisecond * 200)

	// Output:
	// Interval 1
}

// Async与Await
func Example_asyncAwait() {
	promise.Async(func() {
		fmt.Println("Macrotask 1")
	})

	p := promise.New(func(resolve, reject func(v any)) (err any) {
		resolve("hello world")
		return
	})
	v, err := promise.Await(p, 1000)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(v.(string))

	time.Sleep(time.Millisecond * 50)

	// Output:
	// Macrotask 1
	// hello world
}
