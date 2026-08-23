# AGENTS.md

本指南为在此仓库工作的代码 Agent 提供代码库指引。本仓库是一个 Go 库：`Promise-Go`，行为符合 `Promises/A+` 规范，并参考 `ES/Promise`，尽可能模拟了 JavaScript 事件循环中 Promise 的行为。

## 常用命令

- 运行全部测试（与 CI 一致）：`go test -v ./...`
- 运行测试（静默）：`go test ./...`
- 运行单个测试：`go test -run TestXxx ./...`（测试位于 `test/`、`aplus_test/` 目录及根目录 `example_test.go`）
- 构建：`go build ./...`
- 静态检查：`go vet ./...`
- CI 配置：`.github/workflows/go-test.yml`，Ubuntu + `go-version-file: go.mod`（go 1.25），仅执行 `go test -v ./...`；无 Makefile、无额外 lint 配置（`go fmt` 即可）。

## 架构概览

单一 Go 包 `promise`（module `github.com/TikaFlow/promise-go`），无子包。核心难点是事件循环调度与 Promises/A+ 状态机，改动某个功能往往需要同时修改多个文件。

> 暂未实现与其他 `Promise`(thenable) 实现的互操作。

### EventLoop 与事件循环

`StartEventLoop(workerCount)` 创建并启动一个 `EventLoop`，模拟 JS 事件循环：执行每个宏任务前总是先清空微任务队列（顺序：清空微队列 → 执行一个宏任务 → 清空微队列 ……）。所有 Promise 都绑定到某个 EventLoop（字段 `prom.eventLoop`），且必须通过 `el.Stop()` 关闭。

事件循环由依赖 `github.com/TikaFlow/worker-pool` 驱动，内部含三个 worker-pool：

- `looper`（1 个 worker）：运行 `run()` 主循环，消费微任务/宏任务队列
- `scheduler`（1 个 worker）：运行 `timeline.run()`，负责定时器调度
- `worker`（workerCount 个 worker）：执行 `Async`/`pushTask` 等通用异步任务

### 文件职责

- `core.go`：包文档与术语约定、状态常量（`Pending`/`Fulfilled`/`Rejected`）、回调类型（`ThenCallback`/`CatchCallback`/`FinallyCallback`/`Executor`）、`StartEventLoop` 的初始化和 worker-pool 组装
- `promise.go`：`Promise` 结构体与公开方法（`State`/`Done`/`Value`/`Reason`/`Then`/`Catch`/`Finally`/`String`）
- `function.go`：状态机核心 —— `resolvePromise`/`rejectPromise`/`flushHandlers`。Promises/A+ 的 2.3.x 解析流程和回调入微队列都在这里，改动状态机语义先看这个文件
- `eventloop.go`：`EventLoop` 及绝大多数公开 API —— 批量（`All`/`AllSettled`/`Race`/`Any`/`Some`）、迭代（`Map`/`Filter`/`Each`/`Reduce`）、定时器（`SetTimeout`/`SetInterval`/`ClearTimeout`/`ClearInterval`/`Delay`）、工具（`Async`/`Await`/`Try`/`Resolve`/`Reject`/`NewPromise`/`PromiseWithResolvers`/`QueueMicrotask`/`Timeout`）、钩子注册（`On`/`Off`）、`Stop`
- `timeline.go`：`timeLine` 定时器调度器，用按时排序的切片 + `time.Timer` + 通道，将到期任务推送到宏任务队列
- `hook.go`：`HookType` 常量与钩子注册表 `promiseHooks`
- `promise_error.go`：错误类型（`TypeError`/`RangeError`/`TimeoutError`/`AggregateError`/`UnexpectedError`）

### 状态机要点

- Promise 三态：`Pending` → `Fulfilled`/`Rejected`，不可逆；用 `sync.Once`（字段 `resolved`）保证 resolve/reject 仅生效一次
- 已决值是另一个 Promise 时采用其状态（规范 2.3.2）；解决值是自身则抛 `TypeError`（循环检测，规范 2.3.1）
- `Then` 返回的新 Promise 状态由回调执行结果决定；回调返回 err 或 panic → 新 Promise 被拒绝（规范 2.2.7.2）
- 微任务（Promise 回调）优先于宏任务（定时器）执行

## 测试组织

- `aplus_test/`：Promises/A+ 官方合规套件移植（`promises-aplus-tests`，约 209 个叶子用例 + 9 组 N/A skip），用于自检 A+ 合规性；`main_test.go` 的 `TestMain` 创建共享 `el = StartEventLoop(1)` 并在退出时 `Stop()`。详见解包内 `REPORT.md`
- `test/`：库自身功能测试——宏任务队列（`SetTimeout`/`SetInterval`）、ES/Promise 扩展（`All`/`AllSettled`/`Race`/`Any`/`Some`/`Map`/`Filter`/`Each`/`Reduce`）、工具 API（`Await`/`Delay`/`Try`/`Async`/`PromiseWithResolvers`/`Timeout`）、钩子（`On`/`Off`）、状态访问器（`State`/`Value`/`Reason`/`Done`）、事件循环时序（宏/微任务顺序）等非 A+ 部分。两个目录均使用外部测试包名（`promise_test`），点导入 `. "github.com/TikaFlow/promise-go"` 引用主包
- 根目录 `example_test.go` 是可运行的示例（`Output:` 注释即断言）
- 涉及事件循环/时序的测试常依赖 `time.Sleep`，执行结果与 goroutine 繁忙程度相关；单独运行某个测试时需留意事件循环 worker 数量与延时
