package promise

import "sync"

const (
	// queueMinBufSize pop 通道缓冲区的最小容量
	queueMinBufSize = 0x10
	// queueMaxBufSize pop 通道缓冲区的最大容量
	queueMaxBufSize = 0x100000
	// queueMaxRetain 保留节点数的上限
	queueMaxRetain = 0x400000
)

// queueNode 链表的节点：一个元素值 + 指向下一节点的指针
type queueNode[T any] struct {
	value T
	next  *queueNode[T]
}

// Queue 是一个"无限容量"的 FIFO 队列，对外表现为一个只读 channel（[Queue.Pop]）。
//
// Queue 必须通过 [NewQueue] 创建；不再使用时调用 [Queue.Close] 停止内部 goroutine。
type Queue[T any] struct {
	mu        sync.Mutex
	head      *queueNode[T] // 下一个待读节点
	tail      *queueNode[T] // 下一个可写节点
	last      *queueNode[T] // 链表真正的末尾节点（next 为 nil）
	nodeCount int           // 链上节点总数
	bufCap    int           // pop 通道容量（构造时固定）
	retain    int           // 保留节点上限
	closed    bool
	ch        chan T        // 对外只读通道（有缓冲）
	notify    chan struct{} // 信号通道，用于唤醒 feed goroutine
}

// NewQueue 创建一个"无限容量"的 FIFO 队列。
//
// popBufSize 是 [Queue.Pop] 返回通道的缓冲容量，最小 16、最大 约1M（越界自动收敛）；
// 预分配的节点数和默认保留上限均为 2×popBufSize，也是 [Queue.SetRetain] 的下限。
// 创建后立即启动一个后台 goroutine（feed）负责搬运元素。
func NewQueue[T any](popBufSize int) *Queue[T] {
	if popBufSize < queueMinBufSize {
		popBufSize = queueMinBufSize
	}
	if popBufSize > queueMaxBufSize {
		popBufSize = queueMaxBufSize
	}
	preAlloc := 2 * popBufSize
	q := &Queue[T]{
		bufCap: popBufSize,
		retain: preAlloc,
		ch:     make(chan T, popBufSize),
		notify: make(chan struct{}, 1),
	}
	// 预分配 2×popBufSize 个节点，作为初始复用池
	head := &queueNode[T]{}
	q.head = head
	q.tail = head
	q.last = head
	for i := 1; i < preAlloc; i++ {
		n := &queueNode[T]{}
		q.last.next = n
		q.last = n
	}
	q.nodeCount = preAlloc
	go q.feed()
	return q
}

// Push 将元素追加到队尾，永不阻塞（除非内存不足）。
// 队列已关闭时返回 false 且不追加。
func (q *Queue[T]) Push(v T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return false
	}
	q.tail.value = v
	// 推进 tail：优先复用末尾的空闲节点，否则分配新节点
	if q.tail.next == nil {
		q.tail.next = &queueNode[T]{}
		q.last = q.tail.next
		q.nodeCount++
	}
	q.tail = q.tail.next
	// 非阻塞唤醒 feed；信号可合并，多余信号丢弃是安全的
	select {
	case q.notify <- struct{}{}:
	default:
	}
	return true
}

// Pop 返回只读通道。所有消费者共享同一通道，按 FIFO 顺序读取元素。
// 队列关闭且排空后，该通道会被关闭。
func (q *Queue[T]) Pop() <-chan T {
	return q.ch
}

// Close 关闭队列：此后 [Queue.Push] 返回 false；已入队元素仍可被消费，
// 全部消费后 [Queue.Pop] 返回的通道才会关闭。重复调用返回 false。
func (q *Queue[T]) Close() bool {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return false
	}
	q.closed = true
	q.mu.Unlock()
	// 非阻塞唤醒 feed，让它处理完剩余元素后关闭通道
	select {
	case q.notify <- struct{}{}:
	default:
	}
	return true
}

// SetRetain 设置保留的最大节点数，取值范围为 [2×bufCap, maxRetain]，越界时不生效并返回 false。
// 默认保留上限为 2×bufCap。当链上节点数超过该值时，消费完的节点不再回收，
// 从而把内存控制在 retain 附近；传入值越小，回收越早、内存占用越低，但会相应增加分配频率。
// 调用后下次消费时生效。
func (q *Queue[T]) SetRetain(n int) bool {
	if n < 2*q.bufCap || n > queueMaxRetain {
		return false
	}
	q.mu.Lock()
	q.retain = n
	q.mu.Unlock()
	return true
}

// feed 是内部 goroutine：持续消费链上元素并写入 pop 通道。
//
// 空队列时阻塞在 notify 上；收到信号后重新检查。队列关闭且排空后，
// 关闭 pop 通道并退出。
//
// 向 pop 通道发送时不能持有 q.mu，否则通道写满时会阻塞整个队列。
func (q *Queue[T]) feed() {
	// nextValue 取出队首元素并推进 head，队列为空时返回 (zero, false)。
	nextValue := func() (T, bool) {
		if q.head == q.tail {
			var zero T
			return zero, false
		}
		v := q.head.value
		// 清空引用，避免复用节点滞留旧值
		q.head.value = *new(T)

		next := q.head.next
		if q.nodeCount > q.retain {
			q.nodeCount-- // 超过上限，丢弃该节点以控制内存
		} else {
			// 回收复用：先断开 next 防止成环，再追加到链表末尾
			q.head.next = nil
			q.last.next = q.head
			q.last = q.head
		}
		q.head = next
		return v, true
	}

	for {
		q.mu.Lock()
		v, ok := nextValue()
		if ok {
			q.mu.Unlock()
			q.ch <- v // 可能阻塞，不持锁
			continue
		}
		closed := q.closed
		q.mu.Unlock()
		if closed {
			close(q.ch)
			return
		}
		<-q.notify
	}
}
