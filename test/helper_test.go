package promise_test

import (
	"errors"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 一组用于 errors.Is 比对的哨兵错误（与 panic 抛出的须是同一实例）。
var (
	errOnFulfilledPanic = errors.New("onFulfilled boom")
	errOnRejectedPanic  = errors.New("onRejected boom")
	errExecutorPanic    = errors.New("executor boom")
	errAsyncPanic       = errors.New("async boom")
	errTryPanic         = errors.New("try boom")
)

// mustSettle 等待 promise 已决，至多超时 d。
func mustSettle(t *testing.T, p *Promise, d time.Duration) {
	t.Helper()
	select {
	case <-p.Done():
	case <-time.After(d):
		t.Fatalf("timeout waiting for promise to settle (state=%s)", p.State())
	}
}
