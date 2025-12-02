package promise_test

import (
	"fmt"
	"time"

	"github.com/TikaFlow/promise-go"
)

// 基础用法
func Example_base() {
	p := promise.New(func(resolve func(v any), reject func(v error)) (err error) {
		resolve("hello world")
		return
	})

	p.Then(func(v any) (any, error) {
		fmt.Println(v.(string))
		return nil, nil
	}, nil)

	time.Sleep(time.Millisecond * 50)

	// Output:
	// hello world
}

// 链式调用
func Example_chain() {
	p := promise.New(func(resolve func(v any), reject func(v error)) (err error) {
		resolve("hello world")
		return
	})

	p.Then(func(v any) (any, error) {
		fmt.Println(v.(string))
		return v, nil
	}, nil).Then(func(v any) (any, error) {
		fmt.Println(v.(string))
		return v, nil
	}, nil).Catch(func(err error) (any, error) {
		fmt.Println("Nothing happend here.")
		return nil, nil
	}).Finally(func() (any, error) {
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
		// 模拟耗时任务
		time.Sleep(time.Millisecond * 100)
		fmt.Println("Macrotask 1")
	})

	p := promise.New(func(resolve func(v any), reject func(v error)) (err error) {
		resolve("hello world")
		return
	})
	v, err := promise.Await(p, 1000)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(v.(string))

	time.Sleep(time.Millisecond * 150)

	// Output:
	// hello world
	// Macrotask 1
}
