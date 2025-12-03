# Promise-Go

<div align="center">
    <a href="https://github.com/TikaFlow/promise-go/actions/workflows/go-test.yml"><img src="https://github.com/TikaFlow/promise-go/workflows/Tests/badge.svg"></a>
    <a href="https://pkg.go.dev/github.com/TikaFlow/promise-go"><img src="https://pkg.go.dev/badge/github.com/TikaFlow/promise-go.svg"></a>
    <a href="https://goreportcard.com/report/github.com/TikaFlow/promise-go"><img src="https://goreportcard.com/badge/github.com/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go/search?l=go"><img src="https://img.shields.io/github/languages/top/TikaFlow/promise-go"></a>
    <a href="https://promisesaplus.com/"><img src="https://img.shields.io/badge/Promises-A%2B-brightgreen?labelColor=gold"></a>
    <a href="https://github.com/TikaFlow/promise-go"><img src="https://img.shields.io/github/languages/code-size/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go/commits/master"><img src="https://img.shields.io/github/last-commit/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go/blob/master/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go?tab=MIT-1-ov-file"><img src="https://img.shields.io/github/license/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go/tags"><img src="https://img.shields.io/github/v/tag/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go/issues"><img src="https://img.shields.io/github/issues/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go"><img src="https://img.shields.io/github/stars/TikaFlow/promise-go"></a>
    <a href="https://github.com/TikaFlow/promise-go"><img src="https://img.shields.io/maintenance/active/2025"></a>
</div>

---

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
    promise.SetTimeout(func() {
        // 创建一个成功的 Promise
        p := promise.New(func(resolve, reject func(v any)) (err error) {
            // 模拟异步操作
            promise.SetTimeout(func() {
                resolve("操作成功完成")
            }, 1000)
            return
        }, 0)
        
        // 处理 Promise
        p.Then(func(v any) (any, error) {
            println("成功:", v.(string))
            return v, nil
        }, func(reason error) (any, error) {
            println("失败:", reason)
            return nil, nil
        })
    })

    // 等待事件循环结束，避免 main 函数提前退出
	<-promise.Done()
}
```

## 核心接口

### 完整API

请参考[API文档](https://pkg.go.dev/github.com/TikaFlow/promise-go)获取完整的函数列表和详细说明。

### Promise 接口

```go
// Promise 是一个拥有 then 方法的对象，其行为符合 Promises/A+ 规范
type Promise interface {
    // State 返回 Promise 的当前状态
    State() string
    
    // Done 返回一个通道，当 Promise 状态变为 Fulfilled 或 Rejected 时，该通道会被关闭
    Done() chan struct{}

    // Value 返回 Promise 的结果值
    Value() any

    // Reason 返回 Promise 的拒绝原因
    Reason() error
    
    // Then 方法返回一个新的 Promise，其状态和结果值由回调函数的执行结果决定
    Then(onFulfilled ThenCallback, onRejected CatchCallback) Promise
    
    // Catch 方法返回一个新的 Promise，是 Then(nil, onRejected) 的语法糖
    Catch(onRejected CatchCallback) Promise
    
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
```

#### Promise 组合函数

```go
// All 等待所有 Promise 解决
func All(inputs ...any) Promise

// AllSettled 等待所有 Promise 完成（无论成功失败）
func AllSettled(inputs ...any) Promise

// Any 等待第一个 Promise 解决
func Any(inputs ...any) Promise

// Race 等待第一个 Promise 完成
func Race(inputs ...any) Promise
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

p.
    Then(func(v any) (any, error) {
        println(v.(string))  // 输出: hello world
        return v, nil
    }, nil).
    Then(func(v any) (any, error) {
        println(v.(string))  // 输出: hello world
        return v, nil
    }, nil).
    Catch(func(err error) (any, error) {
        println("捕获到错误:", err)
        return nil, nil
    }).
    Finally(func() (any, error) {
        println("无论成功失败都会执行")
        return nil, nil
    })
```

### 更多示例

更多完整的示例可以参考项目中的示例文件：
- [基础示例](example/base_test.go)
- [进阶示例](example/advance_test.go)

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

默认情况下，事件循环会在空闲一段时间后退出自动退出循环，如果一个异步任务执行时间过长（超过这段空闲时间），将会因事件循环退出而无法正确处理`Promise`。

可以通过以下方式延长：

- 【推荐】获取事件循环操作句柄：将无限延长超时时间（即永不超时），直到手动关闭事件循环。
- 手动延长超时时间：调用`promise.SetTimeout()`或`promise.SetInterval()`时，将根据设置的延迟时间自动延长超时时间（一次有效，到期将恢复默认空闲时间）。

#### 等待 Promise 完成

`<-p.Done()`可以等待`Promise`完成，但需要注意，这将会阻塞当前`goroutine`，直到`Promise`完成（有可能永远是`Pending`）。因此应避免使用，
而是使用`p.Then()`或`p.Catch()`来处理`Promise`的结果，或调用`promise.Await()`等待`Promise`完成。

### 错误处理

#### 错误传播

`Promise`链中的错误会自动传播到下一个拒绝处理程序。与在`JS`中一样，可以在链的末端添加`.Catch()`以捕获任何未处理的错误。

#### 类型断言

由于`Go`的类型系统，在处理`Promise`结果时需要进行类型断言。请确保安全地处理断言可能失败的情况。

>   在同一个`promise`的不同分支中，应尽可能`Resolve`同一类型的已决值（尽管`JS`中允许在不同分支`resolve`不同类型的值，这里也使用`any`类型保留了这一特点），
这样可以保证在后续的`Then`回调中安全地进行类型断言。

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

### 性能考虑

#### 定时器精度

定时器（`SetTimeout`/`SetInterval`）的精度依赖于`Go`运行时和系统调度，不保证毫秒级精确执行及执行顺序。

#### 任务队列大小

大量的微任务或宏任务可能会影响性能，特别是在高并发场景下。请合理使用这些功能。

### 常见陷阱

#### 阻塞事件循环

在`Promise`执行器或回调中执行耗时操作会阻塞整个事件循环，应避免这样做。

#### 嵌套过多

过多的`Promise`嵌套会使代码难以理解和维护。尽量使用链式调用或组合函数来扁平化异步流程。

## 许可证

```
MIT License

Copyright (c) [2025] [兮夏]

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
