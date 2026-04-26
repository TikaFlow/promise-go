/*
Package promise 提供了 [Promise] 的 Golang 实现，其行为符合 [Promises/A+] 规范，
并参考 [ES/Promise] 规范，尽可能模拟了 JavaScript 事件循环中 [ES/Promise] 的行为。

# 注意

为了完美的文档表现，使用了 [DocEventLoop] 和 [DocPromise] 结构体，去除 Doc 前缀即它指代的接口，请勿使用这些类型

与 [ES/Promise] 一样，运行在单个 goroutine 中，因此也“继承”了一些特点：
  - 可以在不同 goroutine 中创建、使用 [Promise]，且能保证逻辑上的有序调用
  - [Promise.Then] 中的回调函数长时间运行将会阻塞其他 [Promise] 实例的运行
  - [DocEventLoop.SetTimeout] 和 [DocEventLoop.SetInterval] 无法保证精确的调度，会受到 goroutine 繁忙的影响

为了统一术语、避免歧义，本包中所有注释，对术语进行以下统一、规范描述：
  - [Pending]: 描述为 [Pending] 或“未决”
  - [Settled]: 描述为 [Settled] 或 “已决”，注意与 [Fulfilled] 区分
  - [Fulfilled]: 描述为 [Fulfilled] 或“解决”
  - [Rejected]: 描述为 [Rejected] 或“拒绝”
  - 除非特别说明，注释中的“返回 err”即代指 err 不为 nil，“未返回 err”即代指 err 为 nil

它还实现了 [fmt.Stringer] 接口，可直接输出 [Promise] 实例的当前状态和结果值。

[Promises/A+]: https://promisesaplus.com/
[ES/Promise]: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise
*/
package promise

// Pending 表示 [Promise] 初始状态，其状态待定，可能转换为 [Fulfilled] 或 [Rejected]
const Pending = "pending"

// Settled [Promise] 已经确定结果，状态不能再发生变化
//
// 注意：这是一个概念上的状态，实现中不能使用此状态，必须明确为 [Fulfilled] 或 [Rejected]
const Settled = "[THIS_SHOULD_NEVER_BE_USED]"

// Fulfilled 表示 [Promise] 已解决，状态不能再发生变化
const Fulfilled = "fulfilled"

// Rejected 表示 [Promise] 已拒绝，状态不能再发生变化
const Rejected = "rejected"

// ThenCallback [Promise] 解决/[Fulfilled] 时的回调函数，它接收 [Promise] 的解决值作为参数
//
// [Promise.Then] 返回的 [Promise] 将以 v 为解决值，
// 但如果返回 err，那么该 [Promise] 将会被拒绝，且拒绝理由为 err
type ThenCallback func(any) (v any, err error)

// CatchCallback [Promise] 拒绝/[Rejected] 时的回调函数，它接收 [Promise] 的拒绝理由作为参数
//
// 返回值与 [ThenCallback] 相同
type CatchCallback func(error) (v any, err error)

// FinallyCallback [Promise] 已决/[Settled] 时的回调函数
// 它不接收任何参数，通常也不需要返回任何值（将被忽略），因为新的 [Promise] 将沿用原 [Promise] 的状态和结果
//
// 特殊情况：
//   - 返回值 v 是一个已拒绝的 [Promise] 实例，将以同样的理由拒绝新 [Promise]
//   - 执行中报错 err（返回），将以同样的理由拒绝新 [Promise]
type FinallyCallback func() (v any, err error)

// Executor [Promise] 构造函数的执行器，代表 [Promise] 需要执行的异步任务
//
// 它接收两个参数：resolve 和 reject，分别用于在合适的时候解决或拒绝 [Promise]
//
// 执行器函数在 [Promise] 实例化时立即调用（同步），且只能调用一次
//
// 注意：如果给 resolve 函数传递一个 [Promise] 实例 p，那么返回的 [Promise] 将可能是 [resoled but not settled]
// 即：它依然可能是任意状态（因为它跟随 p 的状态），但无法再使用 resolve 或 reject 来修改状态
//
// 如果执行器函数返回 err，则 [Promise] 会被拒绝，且拒绝理由为 err
//
// [resoled but not settled]: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise/Promise#return_value
type Executor func(resolve, reject func(v any)) (err error)

// Promise 一个拥有 then 方法的对象，其行为符合 Promises/A+ 规范，文档请通过 [DocPromise] 查看
type Promise interface {
	State() string
	Done() <-chan struct{}
	Value() any
	Reason() error
	Then(onFulfilled ThenCallback, onRejected CatchCallback) Promise
	Catch(onRejected CatchCallback) Promise
	Finally(onFinally FinallyCallback) Promise
}

// EventLoop 核心接口，定义了 [Promise] 异步相关的API，文档请通过 [DocEventLoop] 查看
type EventLoop interface {
	All(inputs ...any) Promise
	AllSettled(inputs ...any) Promise
	Any(inputs ...any) Promise
	Async(fn func() (any, error)) Promise
	Await(prom any, timeout int64) (v any, err error)
	ClearInterval(id int)
	ClearTimeout(id int)
	Delay(prom any, millis int64) Promise
	Each(it func(item any, index int, arrLen int) any, inputs ...any) Promise
	Filter(filter func(item any) bool, inputs ...any) Promise
	Map(mapper func(item any) any, inputs ...any) Promise
	NewPromise(exec Executor) Promise
	Off(event HookType, key string) bool
	On(event HookType, hook func(p Promise)) string
	PromiseWithResolvers() (Promise, func(any), func(any))
	QueueMicrotask(fn func())
	Race(inputs ...any) Promise
	Reduce(reducer func(acc any, cur any) any, initial any, inputs ...any) Promise
	Reject(reason any) Promise
	Resolve(input any) Promise
	SetInterval(callback func(), millis int64) int
	SetTimeout(callback func(), millis int64) int
	Some(num int, inputs ...any) Promise
	Stop()
	Try(fn func(...any) (any, error), args ...any) Promise
}
