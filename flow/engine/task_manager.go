package engine

import (
	"context"
	"sync"

	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

// TaskFunc Task execution function
type TaskFunc func() error

// TaskManager Task manager, orchestrates parallel node execution
type TaskManager struct {
	maxWorker int
	tasks     chan taskItem
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

type taskItem struct {
	fn   TaskFunc
	done chan error
}

func NewTaskManager(maxWorker int) *TaskManager {
	if maxWorker <= 0 {
		maxWorker = 4
	}
	ctx, cancel := context.WithCancel(context.Background())
	tm := &TaskManager{
		maxWorker: maxWorker,
		tasks:     make(chan taskItem, 64),
		ctx:       ctx,
		cancel:    cancel,
	}
	for i := 0; i < maxWorker; i++ {
		tm.wg.Add(1)
		go tm.worker()
	}
	log.Info("TaskManager started", zap.Int("workers", maxWorker))
	return tm
}

func (tm *TaskManager) worker() {
	defer tm.wg.Done()
	for {
		select {
		case <-tm.ctx.Done():
			return
		case item, ok := <-tm.tasks:
			if !ok {
				return
			}
			err := item.fn()
			item.done <- err
		}
	}
}

// Submit Submit a set of tasks for parallel execution, wait for all
// Returns first error if any
func (tm *TaskManager) Submit(funcs []TaskFunc) error {
	if len(funcs) == 0 {
		return nil
	}
	if len(funcs) == 1 {
		return funcs[0]()
	}

	dones := make([]chan error, len(funcs))
	for i, fn := range funcs {
		done := make(chan error, 1)
		dones[i] = done
		tm.tasks <- taskItem{fn: fn, done: done}
	}

	var firstErr error
	for _, done := range dones {
		if err := <-done; err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Shutdown Shutdown TaskManager
func (tm *TaskManager) Shutdown() {
	tm.cancel()
	close(tm.tasks)
	tm.wg.Wait()
}
