package util

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

const logNum = 10

// TimeWheel 单圈时间轮，只用于特定场景，timer 最大时间不能超过 tick*bucketsNum
type TimeWheel struct {
	tick            int32
	bucketsNum      int32
	bucketsMaxIndex int32
	readerIndex     int32
	buckets         []*bucket
	// 上下文
	ctx context.Context

	// 取消函数
	cancel context.CancelFunc

	logLock      sync.Mutex
	timeWheelLog []*TimeWheelLog
	logIndex     int
}

type TimeWheelLog struct {
	Num       int
	StartTime *time.Time
	EndTime   *time.Time
}

type bucket struct {
	queue *SliceQueue
	lock  *sync.Mutex
}

func (b *bucket) add(timer *Timer) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.queue.Write(timer)
}
func (tw *TimeWheel) addTimer(index int32, timer *Timer) {
	tw.buckets[index].add(timer)
}
func (b *bucket) read() (*Timer, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	read, err := b.queue.Read()
	if err != nil {
		return nil, err
	}
	return read.(*Timer), nil
}
func (b *bucket) len() int {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.queue.Len()
}

type Timer struct {
	C       <-chan bool
	c       chan<- bool
	isClose int32
}

func (t *Timer) run() {
	if atomic.CompareAndSwapInt32(&t.isClose, 0, 1) {
		close(t.c)
	}
}
func (t *Timer) Close() {
	if atomic.CompareAndSwapInt32(&t.isClose, 0, 1) {
		close(t.c)
	}
}

func (tw *TimeWheel) GetLog() []*TimeWheelLog {
	tw.logLock.Lock()
	defer tw.logLock.Unlock()
	result := make([]*TimeWheelLog, len(tw.timeWheelLog))
	copy(result, tw.timeWheelLog)
	return result
}

func (tw *TimeWheel) NewTimer(tickSeconds int32) *Timer {
	index := tickSeconds / tw.tick
	y := tickSeconds % tw.tick
	if y > 0 {
		index = index + 1
	}

	c := make(chan bool, 1)
	timer := &Timer{C: c, c: c}

	// 原子读 readerIndex，避免与 scheduler 的数据竞争
	ri := atomic.LoadInt32(&tw.readerIndex)
	vIndex := index + ri
	if vIndex >= tw.bucketsNum {
		vIndex = vIndex - tw.bucketsNum
	}
	tw.addTimer(vIndex, timer)

	// 检查 scheduler 是否在我们放置 timer 期间已经处理了目标 bucket
	newRi := atomic.LoadInt32(&tw.readerIndex)
	if ri != newRi && isBetween(ri, newRi, vIndex) {
		// 目标 bucket 已被越过，立即触发
		timer.run()
	}

	return timer
}

// isBetween 判断 x 是否在半开区间 (start, end] 内（支持循环区间）
func isBetween(start, end, x int32) bool {
	if start < end {
		return x > start && x <= end
	}
	// wraparound: (start, mod-1] ∪ [0, end]
	return x > start || x <= end
}

func (tw *TimeWheel) getBucketsByIndex(index int32) *bucket {
	return tw.buckets[index]
}

func (tw *TimeWheel) addLog(num int, startTime *time.Time, endTime *time.Time) {
	tw.logLock.Lock()
	defer tw.logLock.Unlock()

	if tw.logIndex >= logNum {
		tw.logIndex = 0
	}
	tw.timeWheelLog[tw.logIndex] = &TimeWheelLog{Num: num, StartTime: startTime, EndTime: endTime}
	tw.logIndex++
}

func (tw *TimeWheel) scheduler() {
	index := atomic.LoadInt32(&tw.readerIndex)
	sq := tw.getBucketsByIndex(index)
	startTime := time.Now()
	num := sq.len()
	for {
		tm, err := sq.read()
		if err != nil {
			if index >= tw.bucketsMaxIndex {
				atomic.StoreInt32(&tw.readerIndex, 0)
			} else {
				atomic.StoreInt32(&tw.readerIndex, index+1)
			}
			break
		} else {
			tm.run()
		}
	}
	if num > 0 {
		endTime := time.Now()
		tw.addLog(num, &startTime, &endTime)
	}
}
func (tw *TimeWheel) Start() {
	ticker := time.NewTicker(time.Duration(tw.tick) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			tw.scheduler()
		case <-tw.ctx.Done():
			return
		}
	}
}
func (tw *TimeWheel) Stop() {
	tw.cancel()
}

// NewTimeWheel 单圈时间轮，只用于特定场景，timer 最大时间不能超过 tick*bucketsNum
func NewTimeWheel(tickSeconds int32, bucketsNum int32) *TimeWheel {
	timeWheel := &TimeWheel{tick: tickSeconds, bucketsNum: bucketsNum, bucketsMaxIndex: bucketsNum - 1}
	timeWheel.ctx, timeWheel.cancel = context.WithCancel(context.Background())
	timeWheel.buckets = make([]*bucket, bucketsNum)
	for i := 0; i < int(bucketsNum); i++ {
		timeWheel.buckets[i] = &bucket{queue: new(SliceQueue), lock: new(sync.Mutex)}
	}
	timeWheel.timeWheelLog = make([]*TimeWheelLog, logNum)
	timeWheel.logIndex = 0
	return timeWheel
}
