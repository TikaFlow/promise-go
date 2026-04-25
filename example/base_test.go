package promise_test

import (
	"fmt"
	"time"

	"github.com/TikaFlow/promise-go"
)

// 基础用法
func Example_base() {
	el := promise.StartEventLoop(1)
	defer el.Stop()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("hello world")
		return nil
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
	el := promise.StartEventLoop(1)
	defer el.Stop()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("hello world")
		return nil
	})

	p.
		Then(func(v any) (any, error) {
			fmt.Println(v.(string))
			return v, nil
		}, nil).
		Then(func(v any) (any, error) {
			fmt.Println(v.(string))
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

	time.Sleep(time.Millisecond * 50)
	// Output:
	// hello world
	// hello world
	// Finally.
}

// 微队列
func Example_queueMicrotask() {
	el := promise.StartEventLoop(1)
	defer el.Stop()
	el.QueueMicrotask(func() {
		fmt.Println("Microtask 1")
	})
	el.QueueMicrotask(func() {
		fmt.Println("Microtask 2")
	})

	time.Sleep(time.Millisecond * 50)
	// Output:
	// Microtask 1
	// Microtask 2
}

// 宏任务
func Example_macroTask() {
	el := promise.StartEventLoop(1)
	defer el.Stop()
	id1 := el.SetTimeout(func() {
		fmt.Println("Timeout 1")
	}, 100)

	var id2 int
	id2 = el.SetInterval(func() {
		el.ClearTimeout(id1)
		el.ClearInterval(id2)
		fmt.Println("Interval 1")
	}, 50)

	time.Sleep(time.Millisecond * 200)
	// Output:
	// Interval 1
}

// Async与Await
func Example_asyncAwait() {
	el := promise.StartEventLoop(1)
	defer el.Stop()
	el.Async(func() (any, error) {
		// 模拟耗时任务
		time.Sleep(time.Millisecond * 100)
		fmt.Println("Macrotask 1")
		return nil, nil
	})

	p := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("hello world")
		return nil
	})
	v, err := el.Await(p, 100)
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
