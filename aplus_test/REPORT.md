# aplus-test — Promises/A+ 官方合规套件移植报告

本目录将官方 `promises-aplus-tests`（JS，13 个文件共 872 用例）移植为 Go 测试，供 `promise-go` 库自检合规性。与 `test/`（手写测试）并存、不替换。

## 运行

```bash
go test -v ./aplus_test/...
go test -v ./...   # 与 CI 一致的完整运行
```

## 结果统计（实测 `go test -count=1 ./aplus_test/...`）

- **可移植 PASS：209 个叶子用例**（含「语义等价改写 adjusted」，见下）
- **N/A SKIP：9 组，对应官方 672 个用例**（每类跳过原因见下表）
- **FAIL：0**

> 注：以上按「叶子测试」计数（真正执行断言的用例，经 `t.Run` 嵌套展开）。13 个顶层测试函数 `TestAplus2_x_x` 各对应官方一个条款文件；`go test -v` 输出的 PASS 行数含中间 `t.Run` 分组节点，故偏多（总节点 267）。

## 适配器（adapter）

官方套件调用 `{ resolved, rejected, deferred }`，映射如下：

| 官方 | 本项目 |
|---|---|
| `resolved(value)` | `el.Resolve(value)` |
| `rejected(reason)` | `el.Reject(reason)` |
| `deferred()` | `el.PromiseWithResolvers()` → `(p, resolve, reject)` |

## 逐条款映射

| 条款 | 官方用例 | 可移植 | N/A | 说明 |
|---|---|---|---|---|
| 2.1.2 已决后不迁移状态 | 6 | 6 | 0 | resolve 后 reject 被忽略（`sync.Once`） |
| 2.1.3 已决后不迁移状态 | 6 | 6 | 0 | 镜像 |
| 2.2.1 非函数处理器忽略 | 20 | 4 | 16 | 仅 `nil` 可表示非函数；false/5/object/array 无 Go 对应物 |
| 2.2.2 onFulfilled 调用时机 | 11 | 11 | 0 | after/with-value/not-before/once；含 never（超时守护） |
| 2.2.3 onRejected 调用时机 | 11 | 11 | 0 | 镜像 |
| 2.2.4 异步/clean-stack | 16 | 16 | 0 | **adjusted**：不字面断言 clean-stack，改为断言异步执行 + 微任务优先于宏任务的顺序 |
| 2.2.5 `this` 绑定 | 4 | 0 | 4 | Go 无 this / strict·sloppy 概念 |
| 2.2.6 多次 then / 顺序 / throw 不阻塞 | 30 | 30 | 0 | **部分 adjusted**：sinon stub/spy/callOrder → 自建 order 切片；throw → `return err` |
| 2.2.7 then 返回 promise | 104 | 74 | 30 | throw→err(66)；nil 透传(6)；非函数取值 30 N/A |
| 2.3.1 自解析 TypeError | 2 | 2 | 0 | `errors.As` 断言 `*TypeError` |
| 2.3.2 采纳 promise 状态 | 10 | 10 | 0 | pending 保持 pending / fulfilled 同值 / rejected 同理由 |
| 2.3.3 解析流程（thenable） | 610 | 0 | 610 | 需任意 thenable/getter/访问器副作用/boxed-proto，本库刻意不支持（见 README「设计说明」） |
| 2.3.4 以非对象值 fulfill | 42 | 30 | 12 | undefined(null 收敛) 等原始值；boxed-proto 修饰 12 N/A |
| **合计** | **872** | **≈200** | **672** | |

## 差异处理规则（JS → Go）

1. 非函数处理器 → `nil`（`ThenCallback` 只能为 func 或 nil）。
2. 非 `error` 拒绝理由 → 实现包装为 `*UnexpectedError`（`function.go`），且其 `reason` 字段未导出。断言用 `errors.As` + `reflect.DeepEqual` 穿透比较内层值。
3. `throw` → 处理器返回 `(nil, err)`；`flushHandlers` 据此拒绝。
4. 自解析 → `NewTypeError("Chaining cycle detected for promise")`，`errors.As` 断言 `*TypeError`。
5. 对象严格相等 → 指针哨兵 `==`（`dummy`/`sentinel`/…）；不可比较值用 `reflect.DeepEqual`。
6. `undefined` vs `null`：Go 中 `null→nil`，用哨兵 `jsUndefined` 区分 `undefined`。
7. `this` 绑定、getter 计数、boxed-proto `then` → N/A（Go 无对应语言机制）。

## 有意保留的合规缺口（非 bug）

本库**刻意不支持任意 thenable 采纳**）。官方 2.3.3 中依赖「读取任意对象 `.then`、getter 访问器、原型修改」的 610 个用例因此全部跳过。理由：JS 互操作自洽是因为有唯一内置事件循环，但 Go 无法保证任意 thenable 的运行在同一个事件循环。
