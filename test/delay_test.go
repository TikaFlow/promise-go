package promise_test

import "testing"

// TestDelay 覆盖 EventLoop.Delay 全部分支。
func TestDelay(t *testing.T) {
	t.Parallel()

	t.Run("plain-value", func(t *testing.T) {
		t.Parallel()
		val := "value"

		if _, err := el.Await(el.Delay(val, 50), 40); err == nil {
			t.Fatalf("Expected 'TimeoutError: await timeout', got nil")
		}
		res, err := el.Await(el.Delay(val, 50), 2000)
		if err != nil {
			t.Fatalf("Expected nil, got %v", err)
		}
		if res != val {
			t.Fatalf("Expected '%s', got %s", val, res)
		}
	})

	t.Run("promise", func(t *testing.T) {
		t.Parallel()
		p := el.NewPromise(func(resolve, reject func(v any)) error {
			el.SetTimeout(func() {
				resolve("success")
			}, 50)
			return nil
		})
		// Delay 会先等待 p 已决再追加 millis，故总耗时约 100ms > 90ms → 超时。
		if _, err := el.Await(el.Delay(p, 50), 90); err == nil {
			t.Fatalf("Expected 'TimeoutError: await timeout', got nil")
		}

		p2 := el.NewPromise(func(resolve, reject func(v any)) error {
			el.SetTimeout(func() {
				resolve("success")
			}, 50)
			return nil
		})
		res, err := el.Await(el.Delay(p2, 50), 2000)
		if err != nil {
			t.Fatalf("Expected nil, got %v", err)
		}
		if res != "success" {
			t.Fatalf("Expected 'success', got %s", res)
		}
	})

	t.Run("rejected-promise", func(t *testing.T) {
		t.Parallel()
		rejected := el.Reject("rejected reason")
		p := el.Delay(rejected, 50)

		_, err := el.Await(p, 2000)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}
		if err.Error() != "UnexpectedError: rejected reason" {
			t.Fatalf("Expected 'UnexpectedError: rejected reason', got %v", err)
		}
	})
}
