/*
Package promise 提供了 [Promise] 的 Golang 实现，其行为符合 [Promises/A+] 规范，
并参考 [ES/Promise] 规范，尽可能模拟了 JavaScript 事件循环中 [ES/Promise] 的行为。

与 [ES/Promise] 一样，运行在单个 goroutine 中，因此也“继承”了一些特点：
  - 可以在不同 goroutine 中创建、使用 [Promise]，且能保证逻辑上的有序调用
  - [Promise.Then] 中的回调函数长时间运行将会阻塞其他 [Promise] 实例的运行
  - [EventLoop.SetTimeout] 和 [EventLoop.SetInterval] 无法保证精确的调度，会受到 goroutine 繁忙的影响

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

const (
	// Pending 表示 [Promise] 初始状态，其状态待定，可能转换为 [Fulfilled] 或 [Rejected]
	Pending = "pending"

	// Settled [Promise] 已经确定结果，状态不能再发生变化
	//
	// 注意：这是一个概念上的状态，实现中不能使用此状态，必须明确为 [Fulfilled] 或 [Rejected]
	Settled = "[THIS_SHOULD_NEVER_BE_USED]"

	// Fulfilled 表示 [Promise] 已解决，状态不能再发生变化
	Fulfilled = "fulfilled"

	// Rejected 表示 [Promise] 已拒绝，状态不能再发生变化
	Rejected = "rejected"
)

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

// Promise 一个拥有 then（[Promise.Then]） 方法的对象，其行为符合 Promises/A+ 规范
type Promise interface {
	// State 返回 [Promise] 的当前状态
	State() string

	// Done 返回一个通道，当 [Promise] 状态变为 [Fulfilled] 或 [Rejected] 时，该通道会被关闭
	//
	// 不建议无限等待，因为规范允许 [Promise] 永远保持 [Pending] 状态
	Done() <-chan struct{}

	// Value 返回 [Promise] 的结果值，如果 [Promise] 当前状态不是 [Fulfilled]，则值为 nil
	Value() any

	// Reason 返回 [Promise] 的拒绝理由，如果 [Promise] 当前状态不是 [Rejected]，则值为 nil
	Reason() error

	// Then 返回一个新的 [Promise]，其状态和结果值由 onFulfilled 或 onRejected 回调函数的执行结果决定
	Then(onFulfilled ThenCallback, onRejected CatchCallback) Promise

	// Catch 返回一个新的 [Promise]，其状态和结果值由 onRejected 回调函数的执行结果决定，详见 [MDN]
	//
	// 这是一个语法糖，等价于以下语句：
	//
	// 	promise.Then(nil, onRejected)
	//
	// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/catch
	Catch(onRejected CatchCallback) Promise

	// Finally 返回一个新的 [Promise]，其状态和结果与原 [Promise] 相同，以下情况除外：
	//   - onFinally 抛出异常 e，则以 e 为理由拒绝新 [Promise]
	//   - onFinally 返回一个拒绝的 [Promise] 实例，则以同样的理由拒绝新 [Promise]
	Finally(onFinally FinallyCallback) Promise
}

// EventLoop 核心接口，定义了 [Promise] 异步相关的API
type EventLoop interface {
	// Stop 停止事件循环，将会等待其管理的异步任务完成才会返回
	Stop()

	// SetTimeout 模拟 setTimeout 函数，在指定毫秒数后调用回调函数
	//   - callback 回调函数
	//   - millis 延迟执行的毫秒数，自动修正负值的延迟为0
	//
	// # return
	//
	// 定时器 ID，可通过调用 [EventLoop.ClearTimeout] 函数来清除定时器
	SetTimeout(callback func(), millis int64) int

	// SetInterval 模拟 setInterval 函数，以指定毫秒数为周期，重复调用回调函数
	//   - callback 回调函数
	//   - millis 延迟执行的毫秒数，自动修正负值的延迟为0
	//
	// # return
	//
	// 定时器 ID，可通过调用 [EventLoop.ClearInterval] 函数来清除定时器
	SetInterval(callback func(), millis int64) int

	// ClearTimeout 清除由 [EventLoop.SetTimeout] 函数创建的定时器
	//   - id 定时器ID
	ClearTimeout(id int)

	// ClearInterval 清除由 [EventLoop.SetInterval] 函数创建的定时器
	//   - id 定时器ID
	ClearInterval(id int)

	// NewPromise 创建一个新的 [Promise] 实例
	//   - exec 执行器函数，用于定义 [Promise] 的异步操作
	NewPromise(exec Executor) Promise

	// All 等待所有输入解决
	//   - 如果 inputs 的所有元素都成功解决，新 [Promise] 也会成功解决，且解决值为一个包含所有元素解决值的数组
	//   - 如果任何一个元素被拒绝，新 [Promise] 也会被拒绝，且拒绝理由为第一个被拒绝的元素的拒绝理由
	All(inputs ...any) Promise

	// AllSettled 等待所有 [Promise] 已决（解决或拒绝）
	//   - 新 [Promise] 会在所有 [Promise] 已决后解决，解决值为一个包含所有 [Promise] 完成状态和结果的数组
	AllSettled(inputs ...any) Promise

	// Any 等待 inputs 中第一个解决的元素
	//   - 如果任何一个 [Promise] 解决，新 [Promise] 也会被解决，且解决值为第一个被解决的解决值
	//   - 如果所有 [Promise] 都被拒绝，新 [Promise] 也会被拒绝，且拒绝理由为 [AggregateError]，
	//     其包含所有 [Promise] 拒绝理由的数组，顺序为 [Promise] 数组中的顺序
	Any(inputs ...any) Promise

	// Async 将 fn 作为一个异步任务执行
	//
	// 类似于 `go fn()`，但会在一个专用的 worker-pool 中进行，且能获取返回值
	//
	// # return
	//
	// 一个新的 [Promise] 实例，并在 fn 函数执行完成后变为解决状态，解决值是 fn 的返回值
	// 若 fn 函数抛出异常 err，则 [Promise] 实例会被拒绝，且拒绝理由为 err
	Async(fn func() (any, error)) Promise

	// Await 等待 Promise 已决并获取解决值，可设定超时时间，以免无限等待
	//   - prom 需要等待的 [Promise] 实例，如果不是 [Promise] 实例，则会被直接返回
	//   - timeout 超时时间，单位为毫秒
	//
	// # return
	//
	//   - v 目标 prom 的解决值
	//   - err 拒绝理由，当 err 存在时，代表 [Promise] 被拒绝，此时 v 的值无意义
	Await(prom any, timeout int64) (v any, err error)

	// Delay 返回一个新的 [Promise]，其状态会在延迟时间后被解决
	//   - prom 将会使用的解决值，如果 prom 是 [Promise] 实例，则会等待其已决后才开始计时；
	//     如果是一个已拒绝的 [Promise]，则会立即拒绝新 [Promise]
	//   - millis 延迟时间，单位为毫秒
	Delay(prom any, millis int64) Promise

	// Each 按顺序等待 inputs 的每个元素已决，每个元素的结果会被传递给迭代器 it
	// 如果 it 返回一个 [Promise]，则会等待该 [Promise] 完成后再继续迭代；
	// 如果当前迭代对象是 [Promise]，则会等待其完成后再继续迭代；
	// 迭代过程中遇到任何一个已拒绝 [Promise]，新 Promise 也会以同样的理由被拒绝
	//   - it 对每个元素进行操作的函数，接受三个参数：item（当前元素）、index（当前元素的索引）、arrLen（数组长度）
	//   - inputs 需要迭代的输入
	//
	// 由于迭代器的输出会被丢弃，因此适合副作用操作，如打印日志等
	//
	// # return
	//
	// 一个 [Promise]，其状态将会是：
	//   - 已解决（[Fulfilled]）：如果所有迭代都成功解决，解决值是包含原始输入解决值的数组
	//   - 已拒绝（[Rejected]）：如果迭代过程中任何一个 [Promise] 被拒绝
	Each(it func(item any, index int, arrLen int) any, inputs ...any) Promise

	// Filter 过滤数组中的元素
	//
	// # return
	//
	// 一个新的 [Promise]，其状态将会是：
	//   - 已解决（[Fulfilled]）：如果所有 [Promise] 都成功解决，解决值是过滤后的数组
	//   - 已拒绝（[Rejected]）：如果任何一个 [Promise] 被拒绝
	Filter(filter func(item any) bool, inputs ...any) Promise

	// Map 对输入数组中的每个元素应用一个函数，返回一个新的 [Promise] 数组，
	// 新数组的每个元素都是原数组对应元素应用函数后的结果
	//   - mapper 对每个元素进行映射操作的函数，接受一个参数 item 并返回一个新值
	//   - inputs 将被 mapper 处理的输入
	//
	// # return
	//
	// 一个 [Promise]，其状态将会是：
	//   - 已解决（[Fulfilled]）：如果所有 [Promise] 都成功解决，且每个 [Promise] 的解决值都被 mapper 处理后得到新值。
	//   - 已拒绝（[Rejected]）：如果任何一个 [Promise] 被拒绝
	Map(mapper func(item any) any, inputs ...any) Promise

	// On 绑定一个钩子函数
	//   - event 钩子事件类型，可选值为 [ [OnCreated] | [OnChained] | [OnFulfilled] | [OnRejected] | [OnSettled] ]
	//   - hook 钩子函数，当事件触发时调用，并以触发该事件的 [Promise] 实例作为参数
	//
	// # return
	//
	// 绑定成功返回钩子函数的唯一标识，可用于后续解绑钩子函数，失败返回空字符串
	On(event HookType, hook func(p Promise)) string

	// Off 解绑一个钩子函数
	//   - event 钩子事件类型，可选值为 [ [OnCreated] | [OnChained] | [OnFulfilled] | [OnRejected] | [OnSettled] ]
	//   - key 要解绑的钩子函数的唯一标识，由 [EventLoop.On] 方法返回
	//
	// event 与 key 必须匹配，否则将解绑失败
	//
	// # return
	//
	// 表明解绑是否成功的 bool 值
	Off(event HookType, key string) bool

	// PromiseWithResolvers 创建一个新的 [Promise] 实例，同时返回 resolve 和 reject 函数，
	// 对应于传入给 Promise() 构造函数执行器的两个参数
	//
	// 这使得可以在 [Promise] 外部手动解决或拒绝 [Promise]，详见 [MDN]
	//
	// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/withResolvers#%E6%8F%8F%E8%BF%B0
	PromiseWithResolvers() (Promise, func(any), func(any))

	// QueueMicrotask 将回调函数添加到微任务队列末尾
	QueueMicrotask(fn func())

	// Race 等待第一个 [Promise] 已决，
	// 新 [Promise] 会在第一个 [Promise] 已决后解决或拒绝，解决值或拒绝理由跟随第一个完成的 [Promise]
	Race(inputs ...any) Promise

	// Reduce 对数组中的每个元素应用一个函数 reducer，将其结果累积到 acc 中，最后返回 acc 的值
	//   - reducer 对每个元素进行操作的函数，接受两个参数 acc 和 cur，返回新的 acc
	//   - initial 初始值
	//   - inputs 被操作的数组
	//
	// # return
	//
	// 一个新的 [Promise]，其状态将会是：
	//   - 已解决（[Fulfilled]）：如果所有 [Promise] 都成功解决，且每个 [Promise] 的解决值都被 reducer 处理后得到新值
	//   - 已拒绝（[Rejected]）：如果任何一个 [Promise] 被拒绝
	//
	// 特殊情况：
	//   - 如果 inputs 为空数组，直接返回初始值 initial
	//   - 如果 initial 为 nil，且 inputs 只有一个元素，直接返回该元素
	Reduce(reducer func(acc any, cur any) any, initial any, inputs ...any) Promise

	// Reject 返回一个已拒绝的 [Promise]，拒绝理由为指定值 reason，详见 [MDN]
	//
	// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/reject
	Reject(reason any) Promise

	// Resolve 返回一个已解决的 [Promise]，解决值为指定值 value
	//
	// 如果 value 已经是 [Promise]，则直接返回该 [Promise]，详见 [MDN]
	//
	// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/resolve
	Resolve(input any) Promise

	// Some 等待 inputs 中前 num 个元素解决
	//   - 如果 num 个元素解决，新 [Promise] 也会被解决，且解决值为一个包含所有元素解决值的数组，
	//     其顺序为被解决的顺序
	//   - 如果太多元素被拒绝，以至于新 [Promise] 永远无法满足，那么新 [Promise] 会立即被拒绝，
	//     且拒绝理由为 [AggregateError]，其包含所有元素拒绝理由的数组，顺序为被拒绝的顺序
	//
	// 注意与 [EventLoop.Any] 的不同，不仅是解决值的格式不同，拒绝理由的顺序也不同
	Some(num int, inputs ...any) Promise

	// Try 接受一个任意类型的回调函数，并将其结果封装成一个 [Promise]，详见 [MDN]。
	//   - fn 任意类型的回调函数，接受任意数量的参数，函数返回值格式为 (any, error)
	//   - args 将要传递给 fn 函数的参数列表
	//
	// # return
	//
	// 一个新的 [Promise]，其状态将会是：
	//   - 已解决（[Fulfilled]）：如果 fn 函数返回一个普通值
	//   - 已拒绝（[Rejected]）：如果 fn 函数返回了 err
	//   - 异步解决或拒绝：如果 fn 函数返回一个 [Promise]，新 [Promise] 会吸收该 [Promise] 的状态
	//
	// [MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/try
	Try(fn func(...any) (any, error), args ...any) Promise
}
