package promise

import (
	"sync/atomic"
)

// All [EventLoop.All]
func (el *eventLoopImpl) All(inputs ...any) Promise {
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

// AllSettled [EventLoop.AllSettled]
func (el *eventLoopImpl) AllSettled(inputs ...any) Promise {
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

// Any [EventLoop.Any]
func (el *eventLoopImpl) Any(inputs ...any) Promise {
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

// Async [EventLoop.Async]
func (el *eventLoopImpl) Async(fn func()) Promise {
	if fn == nil {
		return el.Reject(NewTypeError("fn must be a function"))
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		task := func() {
			fn()
			resolve(nil)
		}
		el.pushTask(task)
		return nil
	})
}

// Await [EventLoop.Await]
func (el *eventLoopImpl) Await(prom any, timeout int64) (v any, err error) {
	if timeout <= 0 {
		return nil, NewRangeError("await timeout must be greater than 0")
	}

	prom2, ok := prom.(Promise)
	if !ok {
		return prom, nil
	}

	wait := el.NewPromise(func(resolve, reject func(v any)) error {
		el.SetTimeout(func() {
			reject(NewTimeoutError("await timeout"))
		}, timeout)
		return nil
	})

	select {
	case <-prom2.Done():
		if prom2.State() == Rejected {
			err = prom2.Reason()
		} else {
			v = prom2.Value()
		}
	case <-wait.Done():
		if wait.State() == Rejected {
			err = wait.Reason()
		} else {
			v = wait.Value()
		}
	}

	return
}

// Each [EventLoop.Each]
func (el *eventLoopImpl) Each(it func(item any, index int, arrLen int) any, inputs ...any) Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}
	if len(inputs) == 0 {
		return el.Resolve(make([]any, 0))
	}
	if it == nil {
		return el.Reject(NewTypeError("nil is not a function"))
	}

	prom := el.Resolve("start")
	arrLen := len(inputs)
	result := make([]any, arrLen)
	for index, item := range inputs {
		prom = prom.
			Then(func(any) (any, error) {
				return item, nil
			}, nil).
			Then(func(v any) (any, error) {
				result[index] = v
				return it(v, index, arrLen), nil
			}, nil)
	}

	return prom.
		Then(func(any) (any, error) {
			return result, nil
		}, nil)
}

// Delay [EventLoop.Delay]
func (el *eventLoopImpl) Delay(prom any, millis int64) Promise {
	return el.NewPromise(func(resolve, reject func(v any)) error {
		el.Resolve(prom).Then(func(v2 any) (any, error) {
			el.SetTimeout(func() {
				resolve(v2)
			}, millis)
			return nil, nil
		}, func(r error) (any, error) {
			reject(r)
			return nil, nil
		})
		return nil
	})
}

// Filter [EventLoop.Filter]
func (el *eventLoopImpl) Filter(filter func(item any) bool, inputs ...any) Promise {
	return el.Map(func(item any) any {
		return el.All(item, filter(item))
	}, inputs...).
		Then(func(v any) (any, error) {
			values := v.([]any)
			result := make([]any, 0)
			for _, item := range values {
				tuple := item.([]any)
				if tuple[1].(bool) {
					result = append(result, tuple[0])
				}
			}
			return result, nil
		}, nil)
}

// Map [EventLoop.Map]
func (el *eventLoopImpl) Map(mapper func(item any) any, inputs ...any) Promise {
	if inputs == nil {
		return el.Reject(NewTypeError("nil is not iterable"))
	}
	if len(inputs) == 0 {
		return el.Resolve(make([]any, 0))
	}
	if mapper == nil {
		return el.Reject(NewTypeError("nil is not a function"))
	}

	result := make([]any, len(inputs))
	for index, item := range inputs {
		result[index] = el.Resolve(item).Then(func(v any) (any, error) {
			return mapper(v), nil
		}, nil)
	}

	return el.All(result...)
}

// PromiseWithResolvers [EventLoop.PromiseWithResolvers]
func (el *eventLoopImpl) PromiseWithResolvers() (Promise, func(any), func(any)) {
	var resolve, reject func(any)
	p := el.NewPromise(func(res, rej func(v any)) error {
		resolve = res
		reject = rej
		return nil
	})
	return p, resolve, reject
}

// Race [EventLoop.Race]
func (el *eventLoopImpl) Race(inputs ...any) Promise {
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

// Reduce [EventLoop.Reduce]
func (el *eventLoopImpl) Reduce(reducer func(acc any, cur any) any, initial any, inputs ...any) Promise {
	init := el.Resolve(initial)

	if len(inputs) == 0 {
		return init
	}

	return init.
		Then(func(v any) (any, error) {
			if v == nil && len(inputs) == 1 {
				return inputs[0], nil
			}

			acc := v
			result := el.Each(func(item any, index int, arrLen int) any {
				acc = reducer(acc, item)
				return nil
			}, inputs...).
				Then(func(v any) (any, error) {
					return acc, nil
				}, nil)
			return result, nil
		}, nil)
}

// Reject [EventLoop.Reject]
func (el *eventLoopImpl) Reject(reason any) Promise {
	return el.NewPromise(func(resolve, reject func(v any)) error {
		reject(reason)
		return nil
	})
}

// Resolve [EventLoop.Resolve]
func (el *eventLoopImpl) Resolve(value any) Promise {
	if prom, ok := value.(Promise); ok {
		return prom
	}

	return el.NewPromise(func(resolve, reject func(v any)) error {
		resolve(value)
		return nil
	})
}

// Some [EventLoop.Some]
func (el *eventLoopImpl) Some(num int, inputs ...any) Promise {
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

// Try [EventLoop.Try]
func (el *eventLoopImpl) Try(fn func(...any) (any, error), args ...any) Promise {
	return el.NewPromise(func(resolve, reject func(v any)) error {
		if fn == nil {
			return NewTypeError("Promise executor must be a function")
		}

		result, err := fn(args...)
		if err != nil {
			reject(err)
			return nil
		}
		resolve(result)
		return nil
	})
}
