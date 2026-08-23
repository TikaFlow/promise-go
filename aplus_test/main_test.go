package aplus_test

import (
	"os"
	"testing"

	. "github.com/TikaFlow/promise-go"
)

// el 是本包共享的事件循环，镜像 test/main_test.go 的约定。
// 用 StartEventLoop(1)：looper 固定为 1（见 core.go），worker 数仅影响未被本套件使用的 Async，
// 取 1 使微任务时序最确定。
var el *EventLoop

func TestMain(m *testing.M) {
	el = StartEventLoop(1)
	defer el.Stop()

	os.Exit(m.Run())
}
