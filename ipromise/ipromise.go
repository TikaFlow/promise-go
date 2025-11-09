package ipromise

const (
	// Pending 表示 Promise 初始状态，等待被解决或拒绝
	Pending = "pending"

	// Fulfilled 表示 Promise 已成功解决
	Fulfilled = "fulfilled"

	// Rejected 表示 Promise 已被拒绝
	Rejected = "rejected"
)

// ThenCallback 是 Promise 已决时的回调函数（成功或失败）
type ThenCallback func(any) (v any, err error)

// FinallyCallback 是 Promise 无论成功或失败都要执行的回调函数
type FinallyCallback func() (v any, err error)

// Executor 是 Promise 构造函数的执行器
type Executor func(resolve, reject func(v any)) (err error)

// Promise 是一个拥有 then 方法的对象，其行为符合 Promise/A+ 规范
type Promise interface {
	// State 返回 Promise 的当前状态
	State() string

	// Result 返回 Promise 的结果值
	Result() any

	// Done 返回一个通道，当 Promise 状态变为 Fulfilled 或 Rejected 时，该通道会被关闭
	Done() chan struct{}

	// Then 方法返回一个新的 Promise，其状态和结果值由 onFulfilled 或 onRejected 回调函数的执行结果决定
	Then(onFulfilled, onRejected ThenCallback) Promise

	// Catch 方法返回一个新的 Promise，其状态和结果值由 onRejected 回调函数的执行结果决定
	Catch(onRejected ThenCallback) Promise

	// Finally 方法返回一个新的 Promise，其状态和结果值与原 Promise 相同（除非 onFinally 抛出异常或返回一个 Rejected Promise）
	Finally(onFinally FinallyCallback) Promise
}
