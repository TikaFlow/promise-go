package promise_test

import (
	"sync"
	"testing"
	"time"

	. "github.com/TikaFlow/promise-go"
)

// TestNewQueue 覆盖 NewQueue 的缓冲容量边界。
func TestNewQueue(t *testing.T) {
	t.Parallel()
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

// TestQueue 覆盖 Queue 的 Push/Pop/Close 基础行为分支。
func TestQueue(t *testing.T) {
	t.Parallel()

	t.Run("fifo", func(t *testing.T) {
		t.Parallel()
		q := NewQueue[int](64)
		const n = 3000
		done := make(chan struct{})
		go func() {
			defer close(done)
			for i := range n {
				if !q.Push(i) {
					t.Errorf("push %d 返回 false", i)
					return
				}
			}
		}()
		<-done

		for i := range n {
			v, ok := <-q.Pop()
			if !ok {
				t.Fatalf("第 %d 个元素通道已关闭", i)
			}
			if v != i {
				t.Fatalf("第 %d 个元素 = %d, 期望 %d", i, v, i)
			}
		}
		q.Close()
	})

	t.Run("pop-same-channel", func(t *testing.T) {
		t.Parallel()
		q := NewQueue[int](64)
		defer q.Close()
		ch1 := q.Pop()
		ch2 := q.Pop()
		if ch1 != ch2 {
			t.Fatal("Pop 应始终返回同一个通道")
		}
	})

	t.Run("close", func(t *testing.T) {
		t.Parallel()
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
	})

	t.Run("close-drains", func(t *testing.T) {
		t.Parallel()
		q := NewQueue[int](64)
		const n = 2000
		for i := range n {
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
	})

	t.Run("pop-blocks-until-push", func(t *testing.T) {
		t.Parallel()
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
	})

	t.Run("concurrent", func(t *testing.T) {
		t.Parallel()
		q := NewQueue[int](128)
		defer q.Close()
		const (
			producers = 8
			perEach   = 2000
		)
		total := producers * perEach

		var wg sync.WaitGroup
		for p := range producers {
			wg.Add(1)
			go func(base int) {
				defer wg.Done()
				for i := range perEach {
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
	})

	t.Run("push-never-blocks", func(t *testing.T) {
		t.Parallel()
		q := NewQueue[int](16)
		defer q.Close()
		done := make(chan struct{})
		go func() {
			for i := range 100000 {
				q.Push(i)
			}
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Push 在无消费者时阻塞了")
		}
	})
}

// TestQueueSetReserve 覆盖 Queue.SetReserve 的边界与容量回收行为。
func TestQueueSetReserve(t *testing.T) {
	t.Parallel()

	t.Run("boundaries", func(t *testing.T) {
		t.Parallel()
		q := NewQueue[int](64)
		if q.SetReserve(64) {
			t.Fatal("reserve < 2×bufCap 应返回 false")
		}
		if !q.SetReserve(128) {
			t.Fatal("reserve == 2×bufCap 应返回 true")
		}
		if !q.SetReserve(4_194_304) {
			t.Fatal("reserve == 上限 应返回 true")
		}
		if q.SetReserve(4_194_305) {
			t.Fatal("reserve > 上限 应返回 false")
		}
		q.Close()
	})

	t.Run("reserve-bound", func(t *testing.T) {
		t.Parallel()
		q := NewQueue[int](64)

		const n = 100000
		for i := range n {
			q.Push(i)
		}
		// 全部消费，验证不阻塞
		for range n {
			<-q.Pop()
		}
		q.Close()
	})
}

// TestQueueOverCapacity 覆盖微任务/宏任务/定时器注册与清除各队列在远超原容量时依然不阻塞。
func TestQueueOverCapacity(t *testing.T) {
	t.Parallel()

	t.Run("microtask", func(t *testing.T) {
		t.Parallel()
		el2 := StartEventLoop(1)
		defer el2.Stop()
		const n = 20000 // 远超原 chan 容量 1024*10
		done := make(chan struct{})
		go func() {
			for range n {
				el2.QueueMicrotask(func() {})
			}
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("微任务队列在超容量时阻塞了")
		}
	})

	t.Run("macrotask", func(t *testing.T) {
		t.Parallel()
		el2 := StartEventLoop(1)
		defer el2.Stop()
		const n = 20000
		done := make(chan struct{})
		go func() {
			for range n {
				el2.SetTimeout(func() {}, 100000) // 超长延迟，避免触发执行
			}
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("宏任务队列在超容量时阻塞了")
		}
	})

	t.Run("timeline-task", func(t *testing.T) {
		t.Parallel()
		el2 := StartEventLoop(1)
		defer el2.Stop()
		const n = 20000
		done := make(chan struct{})
		go func() {
			for range n {
				el2.SetTimeout(func() {}, 100000)
			}
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("定时器注册队列在超容量时阻塞了")
		}
	})

	t.Run("timeline-clear", func(t *testing.T) {
		t.Parallel()
		el2 := StartEventLoop(1)
		defer el2.Stop()
		// 注册少量定时器，其 id 供清除复用
		var ids []int
		for range 100 {
			ids = append(ids, el2.SetTimeout(func() {}, 100000))
		}
		const n = 20000
		done := make(chan struct{})
		go func() {
			for i := range n {
				el2.ClearTimeout(ids[i%len(ids)])
			}
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("定时器清除队列在超容量时阻塞了")
		}
	})
}
