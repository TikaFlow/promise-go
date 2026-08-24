package promise

// Each 按顺序等待 inputs 的每个元素已决，每个元素的结果会被传递给迭代器 it
// 如果 it 返回一个 [Promise]，则会等待该 [Promise] 完成后再继续迭代；
// 如果当前迭代对象是 [Promise]，则会等待其完成后再继续迭代；
// 迭代过程中遇到任何一个已拒绝 [Promise]，新 Promise 也会以同样的理由被拒绝
//   - it 对每个元素进行操作的函数，接受三个参数：item（当前元素）、index（当前元素的索引）、arrLen（数组长度）
//   - inputs 需要迭代的输入
//
// 由于迭代器的输出会被丢弃，因此适合副作用操作，如打印日志等
//
// # return
//
// 一个 [Promise]，其状态将会是：
//   - 已解决（[Fulfilled]）：如果所有迭代都成功解决，解决值是包含原始输入解决值的数组
//   - 已拒绝（[Rejected]）：如果迭代过程中任何一个 [Promise] 被拒绝
func (el *EventLoop) Each(it func(item any, index int, arrLen int) any, inputs ...any) *Promise {
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

// Filter 过滤数组中的元素
//
// # return
//
// 一个新的 [Promise]，其状态将会是：
//   - 已解决（[Fulfilled]）：如果所有 [Promise] 都成功解决，解决值是过滤后的数组
//   - 已拒绝（[Rejected]）：如果任何一个 [Promise] 被拒绝
func (el *EventLoop) Filter(filter func(item any) bool, inputs ...any) *Promise {
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

// Map 对输入数组中的每个元素应用一个函数，返回一个新的 [Promise] 数组，
// 新数组的每个元素都是原数组对应元素应用函数后的结果
//   - mapper 对每个元素进行映射操作的函数，接受一个参数 item 并返回一个新值
//   - inputs 将被 mapper 处理的输入
//
// # return
//
// 一个 [Promise]，其状态将会是：
//   - 已解决（[Fulfilled]）：如果所有 [Promise] 都成功解决，且每个 [Promise] 的解决值都被 mapper 处理后得到新值。
//   - 已拒绝（[Rejected]）：如果任何一个 [Promise] 被拒绝
func (el *EventLoop) Map(mapper func(item any) any, inputs ...any) *Promise {
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

// Reduce 对数组中的每个元素应用一个函数 reducer，将其结果累积到 acc 中，最后返回 acc 的值
//   - reducer 对每个元素进行操作的函数，接受两个参数 acc 和 cur，返回新的 acc
//   - initial 初始值
//   - inputs 被操作的数组
//
// # return
//
// 一个新的 [Promise]，其状态将会是：
//   - 已解决（[Fulfilled]）：如果所有 [Promise] 都成功解决，且每个 [Promise] 的解决值都被 reducer 处理后得到新值
//   - 已拒绝（[Rejected]）：如果任何一个 [Promise] 被拒绝
//
// 特殊情况：
//   - 如果 inputs 为空数组，直接返回初始值 initial
//   - 如果 initial 为 nil，且 inputs 只有一个元素，直接返回该元素
func (el *EventLoop) Reduce(reducer func(acc any, cur any) any, initial any, inputs ...any) *Promise {
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
