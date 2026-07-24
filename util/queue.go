package util

import (
	"errors"
	"sync"
)

var ErrQueueClosed = errors.New("queue closed")

type Queue struct {
	sliceQueue *SliceQueue
	lock       *sync.RWMutex
	waitNum    int32
	flag       chan bool
	closed     bool
}

func (queue *Queue) Offer(value any) error {
	queue.lock.Lock()
	if queue.closed {
		queue.lock.Unlock()
		return ErrQueueClosed
	}
	err := queue.sliceQueue.Write(value)
	if queue.waitNum > 0 {
		queue.waitNum--
		queue.lock.Unlock()
		// 非阻塞发送：如果 waiter 已超时离开，丢弃信号
		select {
		case queue.flag <- true:
		default:
		}
	} else {
		queue.lock.Unlock()
	}
	return err
}

func (queue *Queue) DequeueTimer(timer *Timer) (value any, hasValue bool) {
	for {
		queue.lock.Lock()
		v, err := queue.sliceQueue.Read()
		if err == nil {
			queue.lock.Unlock()
			timer.Close()
			return v, true
		}
		if queue.closed {
			queue.lock.Unlock()
			timer.Close()
			return nil, false
		}

		queue.waitNum++
		queue.lock.Unlock()
		select {
		case fa := <-queue.flag:
			if fa {
				continue
			}
			timer.Close()
			return nil, false
		case <-timer.C:
			queue.lock.Lock()
			v, err := queue.sliceQueue.Read()
			if err == nil {
				queue.lock.Unlock()
				timer.Close()
				return v, true
			}
			if queue.waitNum > 0 {
				queue.waitNum--
			}
			queue.lock.Unlock()
			timer.Close()
			return nil, false
		}
	}
}

func (queue *Queue) Dequeue() (value any, hasValue bool) {
	for {
		queue.lock.Lock()
		v, err := queue.sliceQueue.Read()
		if err == nil {
			queue.lock.Unlock()
			return v, true
		}
		if queue.closed {
			queue.lock.Unlock()
			return nil, false
		}

		queue.waitNum++
		queue.lock.Unlock()
		select {
		case fa := <-queue.flag:
			if fa {
				continue
			}
			return nil, false
		}
	}
}

// NewQueue 创建一个新的 Queue。
func NewQueue() *Queue {
	return &Queue{
		sliceQueue: new(SliceQueue),
		lock:       new(sync.RWMutex),
		flag:       make(chan bool),
	}
}

// Close 关闭队列，唤醒所有等待者。幂等，多次调用安全。
func (queue *Queue) Close() {
	queue.lock.Lock()
	if queue.closed {
		queue.lock.Unlock()
		return
	}
	queue.closed = true
	close(queue.flag)
	queue.lock.Unlock()
}
