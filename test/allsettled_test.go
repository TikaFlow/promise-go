package promise_test

import (
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestAllSettled 覆盖 EventLoop.AllSettled 全部分支。
func TestAllSettled(t *testing.T) {
	t.Parallel()

	t.Run("settled", func(t *testing.T) {
		t.Parallel()
		p1 := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve(1)
			return nil
		})
		p2 := el.NewPromise(func(resolve, reject func(v any)) error {
			reject("error")
			return nil
		})
		p3 := el.NewPromise(func(resolve, reject func(v any)) error {
			resolve(3)
			return nil
		})

		p := el.AllSettled(p1, p2, p3)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		results, ok := p.Value().([]map[string]any)
		if !ok {
			t.Fatalf("Expected []map[string]any type, got %T", p.Value())
		}
		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}
		if results[0]["status"] != Fulfilled || results[0]["value"] != 1 {
			t.Fatalf("Expected first result fulfilled with value 1, got %v", results[0])
		}
		if results[1]["status"] != Rejected || results[1]["reason"].(error).Error() != "UnexpectedError: error" {
			t.Fatalf("Expected second result rejected with reason 'error', got %v", results[1])
		}
		if results[2]["status"] != Fulfilled || results[2]["value"] != 3 {
			t.Fatalf("Expected third result fulfilled with value 3, got %v", results[2])
		}
	})

	t.Run("nil-input", func(t *testing.T) {
		t.Parallel()
		p := el.AllSettled()
		mustSettle(t, p, 2*time.Second)
		if p.State() != Rejected {
			t.Fatalf("expect Rejected, got %s", p.State())
		}
		if got := p.Reason().Error(); got != "TypeError: nil is not iterable" {
			t.Fatalf("Expected error 'TypeError: nil is not iterable', got %s", got)
		}
	})

	t.Run("empty-array", func(t *testing.T) {
		t.Parallel()
		p := el.AllSettled(make([]any, 0)...)
		mustSettle(t, p, 2*time.Second)
		if p.State() != Fulfilled {
			t.Fatalf("expect Fulfilled, got %s", p.State())
		}
		results, ok := p.Value().([]map[string]any)
		if !ok {
			t.Fatalf("Expected []map[string]any type, got %T", p.Value())
		}
		if len(results) != 0 {
			t.Fatalf("Expected empty array, got %d elements", len(results))
		}
	})
}
