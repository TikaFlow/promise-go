package promise_test

import (
	"os"
	"testing"

	. "github.com/TikaFlow/promise-go"
)

var el EventLoop

func TestMain(m *testing.M) {
	el = StartEventLoop(10)
	defer el.Stop()

	os.Exit(m.Run())
}
