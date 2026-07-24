package util

import (
	"errors"
	"io"
	"sync"
)

type readOp int8

const (
	opRead    readOp = -1
	opInvalid readOp = 0
)
const sliceSmallBufferSize = 64

var ErrTooLarge = errors.New("sliceQueue: too large")
var errNegativeRead = errors.New("sliceQueue: reader returned negative count from Read")

const maxInt = int(^uint(0) >> 1)

type SliceQueueSafe struct {
	sliceQueue *SliceQueue
	lock       *sync.Mutex
}

func NewSliceQueueSafe() *SliceQueueSafe {
	return &SliceQueueSafe{sliceQueue: new(SliceQueue), lock: new(sync.Mutex)}
}
func (sqs *SliceQueueSafe) Reset() {
	sqs.lock.Lock()
	defer sqs.lock.Unlock()
	sqs.sliceQueue.Reset()
}
func (sqs *SliceQueueSafe) Write(c any) error {
	sqs.lock.Lock()
	defer sqs.lock.Unlock()
	return sqs.sliceQueue.Write(c)
}
func (sqs *SliceQueueSafe) Read() (any, error) {
	sqs.lock.Lock()
	defer sqs.lock.Unlock()
	return sqs.sliceQueue.Read()
}

type SliceQueue struct {
	buf      []any
	off      int
	lastRead readOp
}

func (b *SliceQueue) tryGrowByReslice(n int) (int, bool) {
	if l := len(b.buf); n <= cap(b.buf)-l {
		b.buf = b.buf[:l+n]
		return l, true
	}
	return 0, false
}

func (b *SliceQueue) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
	b.lastRead = opInvalid
}
func (b *SliceQueue) Len() int { return len(b.buf) - b.off }
func (b *SliceQueue) grow(n int) int {
	m := b.Len()
	if m == 0 && b.off != 0 {
		b.Reset()
	}
	if i, ok := b.tryGrowByReslice(n); ok {
		return i
	}
	if b.buf == nil && n <= sliceSmallBufferSize {
		b.buf = make([]any, n, sliceSmallBufferSize)
		return 0
	}
	c := cap(b.buf)
	if n <= c/2-m {
		copy(b.buf, b.buf[b.off:])
	} else if c > maxInt-c-n {
		panic(ErrTooLarge)
	} else {
		b.buf = growSlice(b.buf[b.off:], b.off+n)
	}
	b.off = 0
	b.buf = b.buf[:m+n]
	return m
}

func growSlice(b []any, n int) []any {
	defer func() {
		if recover() != nil {
			panic(ErrTooLarge)
		}
	}()
	c := len(b) + n
	if c < 2*cap(b) {
		c = 2 * cap(b)
	}
	b2 := append([]any(nil), make([]any, c)...)
	copy(b2, b)
	return b2[:len(b)]
}

func (b *SliceQueue) empty() bool { return len(b.buf) <= b.off }
func (b *SliceQueue) Read() (any, error) {
	if b.empty() {
		b.Reset()
		return 0, io.EOF
	}
	c := b.buf[b.off]
	b.off++
	b.lastRead = opRead
	return c, nil
}
func (b *SliceQueue) Write(c any) error {
	b.lastRead = opInvalid
	m, ok := b.tryGrowByReslice(1)
	if !ok {
		m = b.grow(1)
	}
	b.buf[m] = c
	return nil
}

var poolSliceQueue = &sync.Pool{
	New: func() interface{} {
		return new(SliceQueue)
	},
}

func GetSliceQueue() *SliceQueue {
	sliceQueue := poolSliceQueue.Get().(*SliceQueue)
	sliceQueue.Reset()
	return sliceQueue
}
func FreeSliceQueue(sliceQueue *SliceQueue) {
	poolSliceQueue.Put(sliceQueue)
}
