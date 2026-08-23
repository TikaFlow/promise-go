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
    <a href="https://github.com/TikaFlow/promise-go"><img src="https://img.shields.io/maintenance/active/2026"></a>
</div>

---

`Promise`的`Golang`实现，其行为符合`Promises/A+`规范，并参考`ES/Promise`规范实现，尽可能模拟了`JavaScript`事件循环中`Promise`的行为。

## 目录

- [项目概述](#项目概述)
- [API文档](#API文档)
- [核心概念](#核心概念)
- [快速开始](#快速开始)
- [钩子函数](#钩子函数)
- [注意事项](#注意事项)
- [许可证](#许可证)

## 项目概述

`Promise-Go` 提供了一个完整的异步编程解决方案，具有以下特点：

- 可以在不同 `goroutine` 中创建、使用 `Promise`，且能保证逻辑上的有序调用
- `Promise.Then` 中的回调函数长时间运行将会阻塞其他 `Promise` 实例的运行
- `EventLoop.SetTimeout` 和 `EventLoop.SetInterval` 无法保证精确的调度，会受到 `goroutine` 繁忙的影响

`Promise-Go` 主要包含以下功能：

- 完整的 `Promises/A+` 规范实现，支持链式调用
- 模拟 `JavaScript` 事件循环的微任务队列和宏任务队列
- 批量处理：`All`、`AllSettled`、`Race`、`Any`、`Some`
- 定时器：`SetTimeout`、`SetInterval`、`Delay`
- 迭代方法：`Map`、`Filter`、`Each`、`Reduce`
- 钩子函数：支持在 `Promise` 生命周期的关键节点插入回调
- 错误类型：提供 `TypeError`、`RangeError`、`TimeoutError`、`AggregateError` 等标准错误类型

## API文档

> 完整的API文档：[Promise-Go on Go Packages](https://pkg.go.dev/github.com/TikaFlow/promise-go)

## 核心概念

### 事件循环（EventLoop）

事件循环是 `Promise-Go` 的核心组件，负责调度微任务和宏任务的执行。通过 `StartEventLoop` 启动一个事件循环，它会持续运行直到调用 `Stop` 方法关闭。

> 事件循环的执行顺序为：清空微队列 → 执行一个宏任务 → 清空微队列 → ...

```go
el := promise.StartEventLoop(1)  // 创建包含1个工作线程的事件循环
defer el.Stop()                  // 等待异步任务完成后关闭
```

### Promise 状态

`Promise` 有三种状态：

- **Pending（待定）**：初始状态，可能转换为 `Fulfilled` 或 `Rejected`
- **Fulfilled（已解决）**：操作成功完成，状态不可再变
- **Rejected（已拒绝）**：操作失败，状态不可再变

### 微任务与宏任务

`Promise-Go` 模拟了 JavaScript 的事件循环机制：

- **微任务队列**：包括 `Promise` 回调、`QueueMicrotask` 添加的任务等，具有更高优先级
- **宏任务队列**：包括 `SetTimeout`、`SetInterval` 添加的定时任务等

## 快速开始

### 安装`promise-go`

```bash
go get github.com/TikaFlow/promise-go
```

### 在项目中使用

```go
package main

import (
    "fmt"
    "time"

    "github.com/TikaFlow/promise-go"
)

func main() {
    el := promise.StartEventLoop(1)
    defer el.Stop()

    // 创建 Promise 并链式调用
    p := el.NewPromise(func(resolve, reject func(v any)) error {
        resolve("hello world")
        return nil
    })
    p.Then(func(v any) (any, error) {
        fmt.Println(v.(string))
        return nil, nil
    }, nil)

    // 延时任务
    el.SetTimeout(func() {
        fmt.Println("[A]")
    }, 30)

    time.Sleep(time.Millisecond * 50)
}
```

## 设计说明

> `Promise` 统一用 `any` 承载值，以符合 `Promises/A+` 允许任意已决类型及状态穿透的约定。

> 暂未实现与其他 `Promise`(thenable) 实现的互操作。

## 钩子函数

`EventLoop` 支持在 `Promise` 生命周期的关键节点插入回调：

- **OnCreated**：`Promise` 实例被创建时
- **OnChained**：`Promise` 实例被链式调用（`Then`、`Catch`、`Finally`）时
- **OnFulfilled**：`Promise` 实例解决时
- **OnRejected**：`Promise` 实例拒绝时
- **OnSettled**：`Promise` 实例已决时（无论解决或拒绝）

```go
// 注册钩子
key := el.On(promise.OnCreated, func(p *promise.Promise) {
    fmt.Println("New promise created")
})

// 注销钩子
_ = el.Off(promise.OnCreated, key)
```

## 注意事项

- `Promise` 实例一旦创建，执行器函数会立即同步调用
- `resolve` 和 `reject` 函数一共只能调用一次，多次调用会被忽略
- 如果给 `resolve` 传递一个 `Promise` 实例，返回的 `Promise` 将跟随该实例的状态
- 如果解决值是 `Promise` 本身，会抛出 `TypeError`（防止循环引用）
- `Then` 方法返回的新 `Promise` 状态由回调函数的执行结果决定
- 回调函数抛出异常会导致新 `Promise` 被拒绝
- 微任务（`Promise` 回调）优先于宏任务（定时器）执行

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
