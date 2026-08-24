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

// 2.2.7.2：onFulfilled 内 panic 应作为拒绝理由。
func TestOnFulfilledPanic(t *testing.T) {
	t.Parallel()
	p := el.Resolve("value")
	p2 := p.Then(func(v any) (any, error) {
		panic(errOnFulfilledPanic)
	}, nil)

	mustSettle(t, p2, 2*time.Second)
	if p2.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", p2.State())
	}
	if !errors.Is(p2.Reason(), errOnFulfilledPanic) {
		t.Fatalf("unexpected reason: %v", p2.Reason())
	}
}

// 2.2.7.2 镜像：onRejected 内 panic 应作为拒绝理由。
func TestOnRejectedPanic(t *testing.T) {
	t.Parallel()
	p := el.Reject(errors.New("root reason"))
	p2 := p.Then(nil, func(r error) (any, error) {
		panic(errOnRejectedPanic)
	})

	mustSettle(t, p2, 2*time.Second)
	if p2.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", p2.State())
	}
	if !errors.Is(p2.Reason(), errOnRejectedPanic) {
		t.Fatalf("unexpected reason: %v", p2.Reason())
	}
}

// 非 error 的 panic 值应被包装为 *UnexpectedError。
func TestOnFulfilledPanicWithNonError(t *testing.T) {
	t.Parallel()
	p := el.Resolve("value")
	p2 := p.Then(func(v any) (any, error) {
		panic("a plain string")
	}, nil)

	mustSettle(t, p2, 2*time.Second)
	if p2.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", p2.State())
	}
	if _, ok := p2.Reason().(*UnexpectedError); !ok {
		t.Fatalf("expect *UnexpectedError, got %T", p2.Reason())
	}
}

// `NewPromise` 的 executor 内 panic 应作为拒绝理由。
func TestExecutorPanic(t *testing.T) {
	t.Parallel()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		panic(errExecutorPanic)
	})

	mustSettle(t, p, 2*time.Second)
	if p.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", p.State())
	}
	if !errors.Is(p.Reason(), errExecutorPanic) {
		t.Fatalf("unexpected reason: %v", p.Reason())
	}
}

// 规范 2.3.3（已决后抛异常应被忽略）：executor 先 resolve 再 panic 应保持 Fulfilled。
func TestExecutorPanicAfterResolve(t *testing.T) {
	t.Parallel()
	p := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("ok")
		panic("after resolve")
	})

	mustSettle(t, p, 2*time.Second)
	if p.State() != Fulfilled {
		t.Fatalf("expect Fulfilled, got %s", p.State())
	}
	if p.Value() != "ok" {
		t.Fatalf("unexpected value: %v", p.Value())
	}
}

// Async 的 fn 内 panic 应作为拒绝理由。
func TestAsyncPanic(t *testing.T) {
	t.Parallel()
	p := el.Async(func() (any, error) {
		panic(errAsyncPanic)
	})

	mustSettle(t, p, 2*time.Second)
	if p.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", p.State())
	}
	if !errors.Is(p.Reason(), errAsyncPanic) {
		t.Fatalf("unexpected reason: %v", p.Reason())
	}
}

// Try 的 fn 内 panic 应作为拒绝理由（经 NewPromise 的 executor 捕获）。
func TestTryPanic(t *testing.T) {
	t.Parallel()
	p := el.Try(func(...any) (any, error) {
		panic(errTryPanic)
	})

	mustSettle(t, p, 2*time.Second)
	if p.State() != Rejected {
		t.Fatalf("expect Rejected, got %s", p.State())
	}
	if !errors.Is(p.Reason(), errTryPanic) {
		t.Fatalf("unexpected reason: %v", p.Reason())
	}
}
