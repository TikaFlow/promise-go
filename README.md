# Promise-Go

<div align="center">
    <a href="https://pkg.go.dev/github.com/TikaFlow/promise-go"><img src="https://pkg.go.dev/badge/github.com/TikaFlow/promise-go.svg"></a>
    <a href="https://promisesaplus.com/"><img src="https://img.shields.io/badge/Promises-A%2B-gold?labelColor=aqua"></a>
    <a href="https://goreportcard.com/report/github.com/TikaFlow/promise-go"><img src="https://goreportcard.com/badge/github.com/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go/search?l=go"><img src="https://img.shields.io/github/languages/top/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go/issues"><img src="https://img.shields.io/github/issues/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go"><img src="https://img.shields.io/github/languages/code-size/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go/commits/master"><img src="https://img.shields.io/github/last-commit/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go/blob/master/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go?tab=MIT-1-ov-file"><img src="https://img.shields.io/github/license/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go/tags"><img src="https://img.shields.io/github/v/tag/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go"><img src="https://img.shields.io/github/stars/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go"><img src="https://img.shields.io/maintenance/active/2025"></a>
</div>

`Promise`的`Golang`实现，其行为符合`Promises/A+`规范，并参考`Promise ES`规范实现，尽可能模拟了`JavaScript`事件循环中`Promise`的行为。

## 项目概述

`Promise-Go`提供了一个完整的异步编程解决方案，允许开发者以链式调用的方式处理异步操作，具有以下特点：

- 完全符合`Promises/A+`规范
- 支持`Promise ES`规范中的常用方法
- 模拟`JavaScript`事件循环机制（宏任务和微任务队列）
- 提供丰富的`Promise`组合方法（`All`、`AllSettled`、`Any`、`Race`等）
- 实现`Stringer`接口，支持直接打印`Promise`实例状态
- 提供定时器相关功能（`SetTimeout`、`SetInterval`等）

## 目录

- [项目概述](#项目概述)
- [目录](#目录)
- [快速开始](#快速开始)
- [核心接口与类型](#核心接口与类型)
- [典型示例](#典型示例)
- [注意事项](#注意事项)

## 快速开始

以下是一个简单的示例，展示如何创建和使用`Promise`：

### 安装

使用`Go`模块系统安装`Promise-Go`：

```bash
go get github.com/TikaFlow/promise-go
```

### 导入

```go
import "github.com/TikaFlow/promise-go"
```

### 使用示例

```go
package main

import "github.com/TikaFlow/promise-go"

func main() {
    // 推荐将 Promise 相关代码包装为异步任务，更符合事件循环机制
    promise.Async(func() {
        // 创建一个成功的 Promise
        p := promise.New(func(resolve, reject func(v any)) (err error) {
            // 模拟异步操作
            promise.SetTimeout(func() {
                resolve("操作成功完成")
            }, 1000)
            return
        })
        
        // 处理 Promise
        p.Then(func(v any) (any, error) {
            println("成功:", v.(string))
            return v, nil
        }, func(reason any) (any, error) {
            println("失败:", reason)
            return nil, nil
        })
    })

    // 等待事件循环结束，避免 main 函数提前退出
	<-promise.Done()
}
```

## 核心接口与类型

### 常量

表示`Promise`状态的常量：

```go
// Pending 表示 Promise 初始状态，等待被解决或拒绝
const Pending = "pending"

// Fulfilled 表示 Promise 已成功解决
const Fulfilled = "fulfilled"

// Rejected 表示 Promise 已被拒绝
const Rejected = "rejected"
```

### 回调函数类型

```go
// ThenCallback 是 Promise 已决时的回调函数（成功或失败）
type ThenCallback func(any) (v any, err error)

// FinallyCallback 是 Promise 无论成功或失败都要执行的回调函数
type FinallyCallback func() (v any, err error)

// Executor 是 Promise 构造函数的执行器
type Executor func(resolve, reject func(v any)) (err error)
```

### Promise 接口

```go
// Promise 是一个拥有 then 方法的对象，其行为符合 Promises/A+ 规范
type Promise interface {
    // State 返回 Promise 的当前状态
    State() string
    
    // Result 返回 Promise 的结果值
    Result() any
    
    // Done 返回一个通道，当 Promise 状态变为 Fulfilled 或 Rejected 时，该通道会被关闭
    Done() chan struct{}
    
    // Then 方法返回一个新的 Promise，其状态和结果值由回调函数的执行结果决定
    Then(onFulfilled, onRejected ThenCallback) Promise
    
    // Catch 方法返回一个新的 Promise，是 Then(nil, onRejected) 的语法糖
    Catch(onRejected ThenCallback) Promise
    
    // Finally 方法返回一个新的 Promise，其状态和结果值与原 Promise 相同
    Finally(onFinally FinallyCallback) Promise
}
```

### 主要函数

#### Promise 创建函数

```go
// New 创建一个新的 Promise 实例
func New(exec Executor) Promise

// Resolve 返回一个已解决的 Promise
func Resolve(value any) Promise

// Reject 返回一个已拒绝的 Promise
func Reject(reason any)

// Try 将函数执行结果封装成一个 Promise
func Try(fn func(...any) (any, error), args ...any) Promise

// PromiseWithResolvers 创建一个新的 Promise 并返回 resolve 和 reject 函数
func PromiseWithResolvers() (Promise, func(any), func(any))
```

#### Promise 组合函数

```go
// All 等待所有 Promise 解决
func All(proms ...any) Promise

// AllSettled 等待所有 Promise 完成（无论成功失败）
func AllSettled(proms ...any) Promise

// Any 等待第一个 Promise 解决
func Any(proms ...any) Promise

// Race 等待第一个 Promise 完成
func Race(proms ...any) Promise
```

#### 事件循环相关函数

```go
// SetTimeout 模拟 setTimeout 函数
func SetTimeout(callback func(), millis int64) int

// SetInterval 模拟 setInterval 函数
func SetInterval(callback func(), millis int64) int

// ClearTimeout 清除由 SetTimeout 创建的定时器
func ClearTimeout(id int)

// ClearInterval 清除由 SetInterval 创建的定时器
func ClearInterval(id int)

// QueueMicrotask 将回调函数添加到微任务队列
func QueueMicrotask(fn func())

// Async 将代码包装为异步任务执行
func Async(fn func())

// Await 等待 Promise 完成，并设定超时时间
func Await(prom Promise, timeout int64) (v any, err error)

// EventLoopHandler 返回事件循环句柄，用于关闭事件循环
func EventLoopHandler() io.Closer
```

## 典型示例

### 基础用法

#### 创建 Promise

```go
// 创建一个立即解决的 Promise
p := promise.New(func(resolve, reject func(v any)) (err error) {
    resolve("hello world")
    return
})
```

#### 链式调用

```go
p := promise.New(func(resolve, reject func(v any)) (err error) {
    resolve("hello world")
    return
})

p.Then(func(v any) (any, error) {
    println(v.(string))  // 输出: hello world
    return v, nil
}, nil).Then(func(v any) (any, error) {
    println(v.(string))  // 输出: hello world
    return v, nil
}, nil).Catch(func(err any) (any, error) {
    println("捕获到错误:", err)
    return nil, nil
}).Finally(func() (any, error) {
    println("无论成功失败都会执行")
    return nil, nil
})
```

### 更多示例

更多完整的示例可以参考项目中的示例文件：
- [基础示例](example/base.go)
- [进阶示例](example/advance.go)

## 注意事项

### 事件循环管理

#### 程序退出管理

在长时间运行的程序中，应首先调用`EventLoopHandler()`获取事件循环句柄，以免事件循环因空闲超时而被动退出。
并在程序结束前调用该句柄的`Close()`方法主动结束事件循环。

如果不需要长时间运行，或不需要主动结束事件循环，则可以不调用`EventLoopHandler()`，
而是通过`Done()`获取关闭信号的通道，等待通道关闭。

```go
// 短时间运行的程序，无需主动关闭事件循环
func main() {
    // 创建一个 Promise
    p := promise.New(func(resolve, reject func(v any)) (err error) {
        // 模拟异步操作
        promise.SetTimeout(func() {
            resolve("操作完成")
        }, 2000)
        return
    })

    // 等待事件循环关闭
    <-promise.Done()
    println("Promise 完成")
}

// 长时间运行的程序，建议主动关闭事件循环
func main() {
    // 获取事件循环句柄
    elh := promise.EventLoopHandler()
    // 关闭事件循环
    defer elh.Close()

    // 长时间服务循环
    service()
}
```

#### 超时时间

默认情况下，事件循环的超时时间为512毫秒，并有128毫秒的额外超时保护，即共计640毫秒。如果一个异步任务在640毫秒内未完成，
或空闲时间超过640毫秒，事件循环会被强制退出。

可以通过以下方式延长：

- 【推荐】获取事件循环操作句柄：将无限延长超时时间（即永不超时），直到手动关闭事件循环。
- 手动延长超时时间：调用`promise.SetTimeout()`或`promise.SetInterval()`时，将根据设置的延迟时间自动延长超时时间（一次有效，到期将恢复）。

#### 等待 Promise 完成

`<-p.Done()`可以等待`Promise`完成，但需要注意，这会阻塞当前`goroutine`，直到`Promise`完成（有可能永远是`Pending`）。因此应避免使用，
而是使用`p.Then()`或`p.Catch()`来处理`Promise`的结果。

```go
// 等待 Promise 完成，不推荐
<-p.Done()
println("Promise 完成")

// 推荐使用 Then 处理 Promise 结果
p.Then(func(v any) (any, error) {
    println("Promise 完成:", v)
    return v, nil
}, nil)
```

### 错误处理

#### 错误传播

`Promise`链中的错误会自动传播到下一个拒绝处理程序。与在`JS`中一样，可以在链的末端添加`.Catch()`以捕获任何未处理的错误。

#### 类型断言

由于`Go`的类型系统，在处理`Promise`结果时需要进行类型断言。请确保安全地处理断言可能失败的情况。

```go
p.Then(func(v any) (any, error) {
    // 安全地进行类型断言
    if str, ok := v.(string); ok {
        println("字符串结果:", str)
    } else {
        println("类型错误，不是字符串")
    }
    return v, nil
}, nil)
```

#### 错误信息

`ThenCallback`和`FinallyCallback`回调函数中，如果有报错，请**务必**将错误信息放入在`v`中，并在`err`中放入简单错误信息（也不能为`nil`）。

这是为了支持任意类型的错误信息，即此时应该返回`(detail, summary)`格式的错误，而不是`(nil, err)`。

### 性能考虑

#### 定时器精度

定时器（`SetTimeout`/`SetInterval`）的精度依赖于`Go`运行时和系统调度，不保证毫秒级精确执行及执行顺序。

#### 任务队列大小

大量的微任务或宏任务可能会影响性能，特别是在高并发场景下。请合理使用这些功能。

### 常见陷阱

#### 错误的返回值

在`Then`（及`Catch`、`Finally`）回调中使用不正确的报错方式（即返回`(nil, err)`而不是`(detail, summary)`）会导致后续的`Promise`链接收`nil`。

#### 阻塞事件循环

在`Promise`执行器或回调中执行耗时操作会阻塞整个事件循环，应避免这样做。

#### 嵌套过多

过多的`Promise`嵌套会使代码难以理解和维护。尽量使用链式调用或组合函数来扁平化异步流程。
