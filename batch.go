package promise

import "sync/atomic"

// All 等待所有输入解决
//   - 如果 inputs 的所有元素都成功解决，新 [Promise] 也会成功解决，且解决值为一个包含所有元素解决值的数组
//   - 如果任何一个元素被拒绝，新 [Promise] 也会被拒绝，且拒绝理由为第一个被拒绝的元素的拒绝理由
func (el *EventLoop) All(inputs ...any) *Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}
	if len(inputs) == 0 {
		return el.Resolve(make([]any, 0))
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		results := make([]any, len(inputs))
		var count int32 = 0

		for index, item := range inputs {
			prom := el.Resolve(item)
			prom.Then(func(v any) (any, error) {
				results[index] = v
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == len(inputs) {
					resolve(results)
				}
				return nil, nil
			}, func(reason error) (any, error) {
				reject(reason)
				return nil, nil
			})
		}

		return nil
	})
}

// AllSettled 等待所有 [Promise] 已决（解决或拒绝）
//   - 新 [Promise] 会在所有 [Promise] 已决后解决，解决值为一个包含所有 [Promise] 完成状态和结果的数组
func (el *EventLoop) AllSettled(inputs ...any) *Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}
	if len(inputs) == 0 {
		return el.Resolve(make([]map[string]any, 0))
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		type result struct {
			Status string
			Value  any
			Reason any
		}

		length := len(inputs)
		results := make([]result, length)
		var count int32 = 0
		for index, item := range inputs {
			prom := el.Resolve(item)
			settleData := func() {
				finalResults := make([]map[string]any, length)
				for i, r := range results {
					finalResults[i] = make(map[string]any)
					finalResults[i]["status"] = r.Status
					if r.Status == Fulfilled {
						finalResults[i]["value"] = r.Value
					} else {
						finalResults[i]["reason"] = r.Reason
					}
				}
				resolve(finalResults)
			}
			prom.Then(func(v any) (any, error) {
				results[index] = result{Status: Fulfilled, Value: v}
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == length {
					settleData()
				}
				return nil, nil
			}, func(reason error) (any, error) {
				results[index] = result{Status: Rejected, Reason: reason}
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == length {
					settleData()
				}
				return nil, nil
			})
		}

		return nil
	})
}

// Any 等待 inputs 中第一个解决的元素
//   - 如果任何一个 [Promise] 解决，新 [Promise] 也会被解决，且解决值为第一个被解决的解决值
//   - 如果所有 [Promise] 都被拒绝，新 [Promise] 也会被拒绝，且拒绝理由为 [AggregateError]，
//     其包含所有 [Promise] 拒绝理由的数组，顺序为 [Promise] 数组中的顺序
func (el *EventLoop) Any(inputs ...any) *Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}
	if len(inputs) == 0 {
		err := NewAggregateError(make([]error, 0), "All promises were rejected", "All promises were rejected")
		return el.Reject(err)
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		length := len(inputs)
		reasons := make([]error, length)

		var count int32 = 0

		for index, item := range inputs {
			prom := el.Resolve(item)
			prom.Then(func(v any) (any, error) {
				resolve(v)
				return nil, nil
			}, func(reason error) (any, error) {
				reasons[index] = reason
				if newCount := atomic.AddInt32(&count, 1); int(newCount) == length {
					reject(NewAggregateError(reasons, "All promises were rejected", "All promises were rejected"))
				}
				return nil, nil
			})
		}

		return nil
	})
}

// Race 等待第一个 [Promise] 已决，
// 新 [Promise] 会在第一个 [Promise] 已决后解决或拒绝，解决值或拒绝理由跟随第一个完成的 [Promise]
func (el *EventLoop) Race(inputs ...any) *Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		if len(inputs) == 0 {
			return nil
		}

		for _, item := range inputs {
			prom := el.Resolve(item)
			prom.Then(func(v any) (any, error) {
				resolve(v)
				return nil, nil
			}, func(reason error) (any, error) {
				reject(reason)
				return nil, nil
			})
		}

		return nil
	})
}

// Some 等待 inputs 中前 num 个元素解决
//   - 如果 num 个元素解决，新 [Promise] 也会被解决，且解决值为一个包含所有元素解决值的数组，
//     其顺序为被解决的顺序
//   - 如果太多元素被拒绝，以至于新 [Promise] 永远无法满足，那么新 [Promise] 会立即被拒绝，
//     且拒绝理由为 [AggregateError]，其包含所有元素拒绝理由的数组，顺序为被拒绝的顺序
//
// 注意与 [EventLoop.Any] 的不同，不仅是解决值的格式不同，拒绝理由的顺序也不同
func (el *EventLoop) Some(num int, inputs ...any) *Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}
	if num <= 0 {
		return el.Reject(NewRangeError("num must be greater than 0"))
	}
	if len(inputs) == 0 {
		err := NewAggregateError(make([]error, 0), "All promises were rejected", "All promises were rejected")
		return el.Reject(err)
	}
	if num > len(inputs) {
		return el.Reject(NewRangeError("no enough promises to resolve"))
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		threshold := len(inputs) - num + 1
		values := make([]any, 0, num)
		reasons := make([]error, 0, threshold)
		var resCount int32 = 0
		var rejCount int32 = 0

		for _, item := range inputs {
			prom := el.Resolve(item)
			prom.Then(func(v any) (any, error) {
				values = append(values, v)
				if newCount := atomic.AddInt32(&resCount, 1); int(newCount) == num {
					resolve(values)
				}
				return nil, nil
			}, func(reason error) (any, error) {
				reasons = append(reasons, reason)
				if newCount := atomic.AddInt32(&rejCount, 1); int(newCount) == threshold {
					result := NewAggregateError(reasons, "AggregateError: Too many promises were rejected", "Too many promises were rejected")
					reject(result)
				}
				return nil, nil
			})
		}
		return nil
	})
}
