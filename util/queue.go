package util

import (
	"errors"
	"sync"
)

var ErrQueueClosed = errors.New("queue closed")

type Queue struct {
	sliceQueue *SliceQueue[any]
	lock       *sync.Mutex
	cond       *sync.Cond
	closed     bool
}

func (queue *Queue) Offer(value any) error {
	queue.lock.Lock()
	if queue.closed {
		queue.lock.Unlock()
		return ErrQueueClosed
	}
	err := queue.sliceQueue.Write(value)
	queue.lock.Unlock()
	// 不阻塞，无信号丢失：cond.Signal 唤醒一个等待者，无等待者时直接返回
	queue.cond.Signal()
	return err
}

func (queue *Queue) DequeueTimer(timer *Timer) (value any, hasValue bool) {
	defer timer.Close()

	timedOut := false
	timer.OnFire(func() {
		queue.lock.Lock()
		timedOut = true
		queue.cond.Broadcast()
		queue.lock.Unlock()
	})

	queue.lock.Lock()

	// 快速路径
	if v, err := queue.sliceQueue.Read(); err == nil {
		queue.lock.Unlock()
		return v, true
	}
	if queue.closed {
		queue.lock.Unlock()
		return nil, false
	}

	for !timedOut {
		v, err := queue.sliceQueue.Read()
		if err == nil {
			queue.lock.Unlock()
			return v, true
		}
		if queue.closed {
			queue.lock.Unlock()
			return nil, false
		}
		queue.cond.Wait()
	}
	// 超时后最后尝试一次
	v, err := queue.sliceQueue.Read()
	queue.lock.Unlock()
	if err == nil {
		return v, true
	}
	return nil, false
}

func (queue *Queue) Dequeue() (value any, hasValue bool) {
	queue.lock.Lock()
	defer queue.lock.Unlock()
	for {
		v, err := queue.sliceQueue.Read()
		if err == nil {
			return v, true
		}
		if queue.closed {
			return nil, false
		}
		queue.cond.Wait()
	}
}

// NewQueue 创建一个新的 Queue。
func NewQueue() *Queue {
	q := &Queue{
		sliceQueue: new(SliceQueue[any]),
		lock:       new(sync.Mutex),
	}
	q.cond = sync.NewCond(q.lock)
	return q
}

// Close 关闭队列，唤醒所有等待者。幂等，多次调用安全。
func (queue *Queue) Close() {
	queue.lock.Lock()
	if queue.closed {
		queue.lock.Unlock()
		return
	}
	queue.closed = true
	queue.cond.Broadcast()
	queue.lock.Unlock()
}
