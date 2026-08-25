package promise_test

import (
	"errors"
	"testing"
	"time"
)

// TestCatch 覆盖 Promise.Catch 全部分支。
func TestCatch(t *testing.T) {
	t.Parallel()

	t.Run("handles-rejection", func(t *testing.T) {
		t.Parallel()
		p1 := el.NewPromise(func(resolve, reject func(v any)) error {
			reject("error")
			return nil
		})
		tail := p1.Catch(func(v error) (any, error) {
			return "caught: " + v.Error(), nil
		}).Then(func(v any) (any, error) {
			if v != "caught: UnexpectedError: error" {
				t.Errorf("Expected value 'caught: UnexpectedError: error', got %v", v)
			}
			return nil, nil
		}, nil)
		mustSettle(t, tail, 2*time.Second)
	})

	t.Run("pass-through", func(t *testing.T) {
		t.Parallel()
		p := el.Resolve("success")
		tail := p.Then(func(v any) (any, error) {
			return v.(string) + " passed 1,", nil
		}, nil).Then(func(v any) (any, error) {
			return v.(string) + " passed 2,", nil
		}, func(r error) (any, error) {
			t.Errorf("Promise should not be rejected, got reason: %v", r.Error())
			return nil, nil
		}).Then(func(v any) (any, error) {
			return nil, errors.New(v.(string) + " rejected")
		}, nil).Then(func(v any) (any, error) {
			t.Errorf("Promise should not be fulfilled, got value: %v", v)
			return nil, nil
		}, nil).Catch(func(r error) (any, error) {
			if r.Error() != "success passed 1, passed 2, rejected" {
				t.Errorf("Expected 'success passed 1, passed 2, rejected', got %v", r.Error())
			}
			return nil, nil
		})
		mustSettle(t, tail, 2*time.Second)
	})
}
