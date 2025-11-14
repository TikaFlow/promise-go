package example

import "github.com/TikaFlow/promise-go"

// 基础用法
func BaseExample() {
	p := promise.New(func(resolve, reject func(v any)) (err error) {
		resolve("hello world")
		return
	})

	p.Then(func(v any) (any, error) {
		println(v.(string))
		return nil, nil
	}, nil)

	<-promise.Done()
}

// 链式调用
func ChainExample() {
	p := promise.New(func(resolve, reject func(v any)) (err error) {
		resolve("hello world")
		return
	})

	p.Then(func(v any) (any, error) {
		println(v.(string))
		return v, nil
	}, nil).Then(func(v any) (any, error) {
		println(v.(string))
		return v, nil
	}, nil).Catch(func(err any) (any, error) {
		println("Nothing happend here.")
		return nil, nil
	}).Finally(func() (any, error) {
		println("Finally.")
		return nil, nil
	})

	<-promise.Done()
}

// 宏任务
func MacroTaskExample() {
	// 将代码包装为一个宏任务
	promise.Run(func() {
		println("Macrotask 1")
	})

	<-promise.Done()
}

// 微队列
func MicroQueueExample() {
	promise.QueueMicrotask(func() {
		println("Microtask 1")
	})
	promise.QueueMicrotask(func() {
		println("Microtask 2")
	})

	<-promise.Done()
}

// 延迟执行
func DelayExample() {
	id1 := promise.SetTimeout(func() {
		println("Timeout 1")
	}, 600)

	var id2 int
	id2 = promise.SetInterval(func() {
		promise.ClearTimeout(id1)
		promise.ClearInterval(id2)
		println("Interval 1")
	}, 300)

	<-promise.Done()
}
