package promise_test

import (
	"testing"
)

// 测试Reject方法
func TestReject(t *testing.T) {
	t.Parallel()
	el.Reject("reason").Then(func(v any) (any, error) {
		t.Errorf("Promise.Reject should be rejected, but was fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "UnexpectedError: reason" {
			t.Errorf("Expected value 'UnexpectedError: reason', got '%s'", v.Error())
		}
		return nil, nil
	})
}

// 测试Resolve方法 - 普通值
func TestResolve(t *testing.T) {
	t.Parallel()
	el.Resolve("value").Then(func(v any) (any, error) {
		if v != "value" {
			t.Errorf("Expected value 'value', got %v", v)
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Promise.Resolve should be fulfilled, but was rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试Resolve方法 - Promise对象 - fulfilled状态
func TestResolvePromise(t *testing.T) {
	t.Parallel()
	original := el.NewPromise(func(resolve, reject func(v any)) error {
		resolve("original")
		return nil
	})

	p := el.Resolve(original)

	if p != original {
		t.Errorf("Expected Resolve to return the same Promise instance")
	}

	p.Then(func(v any) (any, error) {
		if v != "original" {
			t.Errorf("Expected value 'original', got %v", v)
		}
		return nil, nil
	}, func(v error) (any, error) {
		t.Errorf("Expected state Fulfilled, got Rejected with %v", v.Error())
		return nil, nil
	})
}

// 测试Resolve方法 - Promise对象 - rejected状态
func TestResolvePromiseRejected(t *testing.T) {
	t.Parallel()
	original := el.NewPromise(func(resolve, reject func(v any)) error {
		reject("error")
		return nil
	})

	p := el.Resolve(original)

	if p != original {
		t.Errorf("Expected Resolve to return the same Promise instance")
	}

	p.Then(func(v any) (any, error) {
		t.Errorf("Expected state Rejected, got Fulfilled with %v", v)
		return nil, nil
	}, func(v error) (any, error) {
		if v.Error() != "UnexpectedError: error" {
			t.Errorf("Expected value 'UnexpectedError: error', got '%s'", v.Error())
		}
		return nil, nil
	})
}
