package util

import (
	"sync"
	"testing"
	"time"
)

func TestQueue_ConcurrentNoDeadlock(t *testing.T) {
	q := NewQueue[int]()
	var wg sync.WaitGroup
	const n = 100000

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			q.Offer(i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			for {
				if _, ok := q.Dequeue(); ok {
					break
				}
			}
		}
	}()
	wg.Wait()
}

func TestQueue_CloseWakeup(t *testing.T) {
	q := NewQueue[int]()
	go func() {
		time.Sleep(10 * time.Millisecond)
		q.Close()
	}()
	_, ok := q.Dequeue()
	if ok {
		t.Error("should return false after close")
	}
}

func TestQueue_DequeueTimer(t *testing.T) {
	tw := NewTimeWheel(1, 256)
	go tw.Start()
	defer tw.Stop()

	q := NewQueue[int]()
	timer := tw.NewCallbackTimer(1) // 1 second
	_, ok := q.DequeueTimer(timer)
	if ok {
		t.Error("should timeout with no data")
	}
}

func TestQueue_DequeueTimerWithData(t *testing.T) {
	tw := NewTimeWheel(1, 256)
	go tw.Start()
	defer tw.Stop()

	q := NewQueue[int]()
	go q.Offer(42)
	timer := tw.NewCallbackTimer(60) // long timeout, data arrives immediately
	v, ok := q.DequeueTimer(timer)
	if !ok || v != 42 {
		t.Errorf("expected 42, got %v (ok=%v)", v, ok)
	}
}

func TestQueue_CloseIdempotent(t *testing.T) {
	q := NewQueue[int]()
	q.Close()
	q.Close() // should not panic
	_, ok := q.Dequeue()
	if ok {
		t.Error("should return false after close")
	}
}

func TestQueue_OfferAfterClose(t *testing.T) {
	q := NewQueue[int]()
	q.Close()
	err := q.Offer(42)
	if err != ErrQueueClosed {
		t.Errorf("expected ErrQueueClosed, got %v", err)
	}
}
