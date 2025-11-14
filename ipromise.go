/*
Package promise 提供了 Promise 的 Golang 实现，其行为符合 Promise A+ 规范，
并参考 Promise ES 规范实现，尽可能模拟了事件循环中 Promise 的行为。

它还实现了 Stringer 接口，可直接打印 Promise 实例的状态和结果值。
*/
package promise

const (
	/*
		Pending 表示 Promise 初始状态，等待被解决或拒绝
	*/
	Pending = "pending"

	/*
		Fulfilled 表示 Promise 已成功解决
	*/
	Fulfilled = "fulfilled"

	/*
		Rejected 表示 Promise 已被拒绝
	*/
	Rejected = "rejected"
)

/*
ThenCallback 是 Promise 已决时的回调函数（成功或失败）。
  - 当它作为 onFulfilled 回调函数时，它接收 Promise 的结果值作为参数。
  - 当它作为 onRejected 回调函数时，它接收 Promise 的拒绝理由作为参数。

当回调函数执行遇到错误时（意味着 Then 方法将会返回一个已拒绝的 Promise 实例），
err 仅代表出错，但应该将错误信息包含在 v 中（以支持任意类型），此时返回的是 (detail, summary) 格式的错误信息。
*/
type ThenCallback func(any) (v any, err error)

/*
FinallyCallback 是 Promise 无论成功或失败都要执行的回调函数。
它不接收任何参数，通常也不需要返回任何值，因为新的 Promise 将沿用原 Promise 的状态和结果值。

特别的：
  - 返回值是一个已拒绝的 Promise 实例，将以同样的理由拒绝新 Promise。
  - 执行中报错，将以同样的理由拒绝新 Promise。

报错和错误格式与 ThenCallback 相同。
*/
type FinallyCallback func() (v any, err error)

/*
Executor 是 Promise 构造函数的执行器。
它接收两个参数：resolve 和 reject，分别用于解决和拒绝 Promise。
执行器函数在 Promise 实例化时立即调用，且只能调用一次。

如果执行器函数抛出异常，则 Promise 会被拒绝，且拒绝理由为该异常。
*/
type Executor func(resolve, reject func(v any)) (err error)

/*
Promise 是一个拥有 then 方法的对象，其行为符合 Promise/A+ 规范。
*/
type Promise interface {
	/*
		State 返回 Promise 的当前状态。
	*/
	State() string

	/*
		Result 返回 Promise 的结果值。
	*/
	Result() any

	/*
		Done 返回一个通道，当 Promise 状态变为 Fulfilled 或 Rejected 时，该通道会被关闭。
	*/
	Done() chan struct{}

	/*
		Then 方法返回一个新的 Promise，其状态和结果值由 onFulfilled 或 onRejected 回调函数的执行结果决定。
	*/
	Then(onFulfilled, onRejected ThenCallback) Promise

	/*
		Catch 方法返回一个新的 Promise，其状态和结果值由 onRejected 回调函数的执行结果决定，详见 [MDN]。

		这是一个语法糖，等价于以下语句：

			promise.Then(nil, onRejected)

		[MDN]: https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise/catch
	*/
	Catch(onRejected ThenCallback) Promise

	/*
		Finally 方法返回一个新的 Promise，其状态和结果值与原 Promise 相同，以下情况除外：
		  - onFinally 抛出异常 e，则以 e 为理由拒绝新 Promise;
		  - onFinally 返回一个拒绝的 Promise 实例，则以同样的理由拒绝新 Promise。
	*/
	Finally(onFinally FinallyCallback) Promise
}
