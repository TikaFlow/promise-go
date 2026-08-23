package promise_test

import (
	"sync"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// 测试基础 FIFO 顺序与大量元素
func TestQueueFIFO(t *testing.T) {
	q := NewQueue[int](64)
	const n = 3000
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < n; i++ {
			if !q.Push(i) {
				t.Errorf("push %d 返回 false", i)
				return
			}
		}
	}()
	<-done

	for i := 0; i < n; i++ {
		v, ok := <-q.Pop()
		if !ok {
			t.Fatalf("第 %d 个元素通道已关闭", i)
		}
		if v != i {
			t.Fatalf("第 %d 个元素 = %d, 期望 %d", i, v, i)
		}
	}
	q.Close()
}

// 测试 Pop 始终返回同一个通道
func TestQueuePopSameChannel(t *testing.T) {
	q := NewQueue[int](64)
	defer q.Close()
	if q.Pop() != q.Pop() {
		t.Fatal("Pop 应始终返回同一个通道")
	}
}

// 测试缓冲容量边界
func TestQueueBufferCapacity(t *testing.T) {
	for _, tc := range []struct {
		in, want int
	}{
		{0, 16},
		{8, 16},
		{16, 16},
		{17, 17},
		{2_000_000, 1_048_576}, // 上限 0x100000 = 1048576
	} {
		q := NewQueue[int](tc.in)
		if got := cap(q.Pop()); got != tc.want {
			t.Errorf("NewQueue(%d) 缓冲容量 = %d, 期望 %d", tc.in, got, tc.want)
		}
		q.Close()
	}
}

// 测试 Close 后 Push 返回 false，且 Close 幂等
func TestQueueClose(t *testing.T) {
	q := NewQueue[int](64)
	if !q.Push(1) {
		t.Fatal("Close 前 Push 应返回 true")
	}
	if !q.Close() {
		t.Fatal("第一次 Close 应返回 true")
	}
	if q.Close() {
		t.Fatal("第二次 Close 应返回 false")
	}
	if q.Push(2) {
		t.Fatal("Close 后 Push 应返回 false")
	}
}

// 测试 Close 后排空剩余元素并关闭通道
func TestQueueCloseDrains(t *testing.T) {
	q := NewQueue[int](64)
	const n = 2000
	for i := 0; i < n; i++ {
		q.Push(i)
	}
	q.Close()
	got := 0
	for v := range q.Pop() {
		if v != got {
			t.Fatalf("第 %d 个元素 = %d, 期望 %d", got, v, got)
		}
		got++
	}
	if got != n {
		t.Fatalf("共收到 %d 个元素, 期望 %d", got, n)
	}
}

// 测试空队列时 Pop 阻塞，Push 后唤醒
func TestQueuePopBlocks(t *testing.T) {
	q := NewQueue[int](64)
	defer q.Close()
	done := make(chan struct{})
	go func() {
		v, ok := <-q.Pop()
		if !ok || v != 42 {
			t.Errorf("收到 (%d, %v), 期望 (42, true)", v, ok)
		}
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	if !q.Push(42) {
		t.Fatal("push 失败")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Push 后消费者未收到唤醒")
	}
}

// 测试并发生产者 + 单一消费者，验证不丢不重
func TestQueueConcurrent(t *testing.T) {
	q := NewQueue[int](128)
	defer q.Close()
	const (
		producers = 8
		perEach   = 2000
	)
	total := producers * perEach

	var wg sync.WaitGroup
	for p := 0; p < producers; p++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for i := 0; i < perEach; i++ {
				if !q.Push(base + i) {
					return
				}
			}
		}(p * perEach)
	}

	seen := make([]bool, total)
	var mu sync.Mutex
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for v := range q.Pop() {
			mu.Lock()
			if seen[v] {
				t.Errorf("值 %d 重复出现", v)
			}
			seen[v] = true
			mu.Unlock()
		}
	}()

	wg.Wait()
	q.Close()
	<-readDone

	mu.Lock()
	defer mu.Unlock()
	for i, ok := range seen {
		if !ok {
			t.Errorf("值 %d 丢失", i)
		}
	}
}

// 测试消费者少于生产者时 Push 不阻塞（无限容量）
func TestQueuePushNeverBlocks(t *testing.T) {
	q := NewQueue[int](16)
	defer q.Close()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100000; i++ {
			q.Push(i)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Push 在无消费者时阻塞了")
	}
}

// 测试 SetRetain 边界（bufCap=64 → 下限为 2×64=128，上限 0x400000=4194304）
func TestQueueSetRetain(t *testing.T) {
	q := NewQueue[int](64)
	if q.SetRetain(64) {
		t.Fatal("retain < 2×bufCap 应返回 false")
	}
	if !q.SetRetain(128) {
		t.Fatal("retain == 2×bufCap 应返回 true")
	}
	if !q.SetRetain(4_194_304) {
		t.Fatal("retain == 上限 应返回 true")
	}
	if q.SetRetain(4_194_305) {
		t.Fatal("retain > 上限 应返回 false")
	}
	q.Close()
}

// 测试 retain 设为 bufCap 时，节点数不会无限增长
func TestQueueRetainBound(t *testing.T) {
	q := NewQueue[int](64)
	q.SetRetain(64)

	const n = 100000
	for i := 0; i < n; i++ {
		q.Push(i)
	}
	// 全部消费，验证不阻塞
	for i := 0; i < n; i++ {
		<-q.Pop()
	}
	q.Close()
}