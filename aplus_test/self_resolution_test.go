package aplus_test

import (
	"testing"

	. "github.com/TikaFlow/promise-go"
)

// 2.3.1: 若 promise 与其解决的 x 是同一对象，则以 TypeError 拒绝。
func TestAplus2_3_1(t *testing.T) {
	t.Run("via return from a fulfilled promise", func(t *testing.T) {
		var promise *Promise
		promise = el.Resolve(dummy).Then(func(any) (any, error) {
			return promise, nil
		}, nil)

		waitTail(t, promise.Then(nil, func(r error) (any, error) {
			assertErrorType(t, r, (*TypeError)(nil))
			return nil, nil
		}))
	})

	t.Run("via return from a rejected promise", func(t *testing.T) {
		var promise *Promise
		promise = el.Reject(dummy).Then(nil, func(error) (any, error) {
			return promise, nil
		})

		waitTail(t, promise.Then(nil, func(r error) (any, error) {
			assertErrorType(t, r, (*TypeError)(nil))
			return nil, nil
		}))
	})
}
