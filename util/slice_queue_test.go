package util

import (
	"io"
	"sync"
	"testing"
)

func TestSliceQueue_BasicWriteRead(t *testing.T) {
	var q SliceQueue[int]
	q.Write(1)
	q.Write(2)
	q.Write(3)

	v, err := q.Read()
	if err != nil || v != 1 {
		t.Fatalf("expected 1, got %v (err=%v)", v, err)
	}
	v, err = q.Read()
	if err != nil || v != 2 {
		t.Fatalf("expected 2, got %v (err=%v)", v, err)
	}
	v, err = q.Read()
	if err != nil || v != 3 {
		t.Fatalf("expected 3, got %v (err=%v)", v, err)
	}
}

func TestSliceQueue_EmptyReadReturnsEOF(t *testing.T) {
	var q SliceQueue[int]
	_, err := q.Read()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestSliceQueue_WriteReadInterleaved(t *testing.T) {
	var q SliceQueue[int]
	for i := 0; i < 100; i++ {
		q.Write(i)
		v, err := q.Read()
		if err != nil || v != i {
			t.Fatalf("expected %d, got %d (err=%v)", i, v, err)
		}
	}
}

func TestSliceQueue_WriteBeyondInitialCap(t *testing.T) {
	var q SliceQueue[int]
	n := 200
	for i := 0; i < n; i++ {
		q.Write(i)
	}
	if q.Len() != n {
		t.Fatalf("len: expected %d, got %d", n, q.Len())
	}
	for i := 0; i < n; i++ {
		v, err := q.Read()
		if err != nil || v != i {
			t.Fatalf("pos %d: expected %d, got %d (err=%v)", i, i, v, err)
		}
	}
}

func TestSliceQueue_WrapAround(t *testing.T) {
	var q SliceQueue[int]
	// Fill to exactly initialCap (64) to force write pointer to wrap
	for i := 0; i < 64; i++ {
		q.Write(i)
	}
	if q.Len() != 64 {
		t.Fatalf("len after fill: expected 64, got %d", q.Len())
	}
	// Read half → read pointer moves to 32
	for i := 0; i < 32; i++ {
		v, _ := q.Read()
		if v != i {
			t.Fatalf("read phase 1: expected %d, got %d", i, v)
		}
	}
	if q.Len() != 32 {
		t.Fatalf("len after half drain: expected 32, got %d", q.Len())
	}
	// Write 32 more → wraps around, writes to indices 0..31
	for i := 64; i < 96; i++ {
		q.Write(i)
	}
	if q.Len() != 64 {
		t.Fatalf("len after re-fill: expected 64, got %d", q.Len())
	}
	// Read remaining 64, should be 32..63 then 64..95 in order
	for i := 32; i < 96; i++ {
		v, err := q.Read()
		if err != nil || v != i {
			t.Fatalf("pos %d: expected %d, got %d (err=%v)", i, i, v, err)
		}
	}
}

func TestSliceQueue_WrapThenGrow(t *testing.T) {
	var q SliceQueue[int]
	// Fill
	for i := 0; i < 64; i++ {
		q.Write(i)
	}
	// Drain 60, only 4 left at tail end
	for i := 0; i < 60; i++ {
		q.Read()
	}
	// Fill another 60 → wraps and triggers grow
	for i := 64; i < 124; i++ {
		q.Write(i)
	}
	if q.Len() != 64 {
		t.Fatalf("len: expected 64, got %d", q.Len())
	}
	// Order: 60,61,62,63,64,65,...,123
	for i := 60; i < 124; i++ {
		v, err := q.Read()
		if err != nil || v != i {
			t.Fatalf("pos %d: expected %d, got %d (err=%v)", i, i, v, err)
		}
	}
}

func TestSliceQueue_Reset(t *testing.T) {
	var q SliceQueue[int]
	for i := 0; i < 100; i++ {
		q.Write(i)
	}
	q.Reset()
	if q.Len() != 0 {
		t.Fatalf("len after reset: expected 0, got %d", q.Len())
	}
	_, err := q.Read()
	if err != io.EOF {
		t.Fatalf("expected io.EOF after reset, got %v", err)
	}
	// Reuse after reset
	q.Write(42)
	v, _ := q.Read()
	if v != 42 {
		t.Fatalf("expected 42 after reset+write, got %d", v)
	}
}

func TestSliceQueue_ResetReusesBuffer(t *testing.T) {
	var q SliceQueue[int]
	q.Write(1)
	capBefore := cap(q.buf)
	q.Reset()
	q.Write(2)
	capAfter := cap(q.buf)
	if capAfter != capBefore {
		t.Fatalf("Reset should retain buffer: cap %d → %d", capBefore, capAfter)
	}
}

func TestSliceQueue_Len(t *testing.T) {
	var q SliceQueue[int]
	if q.Len() != 0 {
		t.Fatalf("initial len: expected 0, got %d", q.Len())
	}
	q.Write(1)
	if q.Len() != 1 {
		t.Fatalf("len after 1 write: expected 1, got %d", q.Len())
	}
	q.Write(2)
	if q.Len() != 2 {
		t.Fatalf("len after 2 writes: expected 2, got %d", q.Len())
	}
	q.Read()
	if q.Len() != 1 {
		t.Fatalf("len after 1 read: expected 1, got %d", q.Len())
	}
	q.Read()
	if q.Len() != 0 {
		t.Fatalf("len after 2 reads: expected 0, got %d", q.Len())
	}
}

func TestSliceQueue_ManyWritesReads(t *testing.T) {
	var q SliceQueue[int]
	const total = 10000
	for i := 0; i < total; i++ {
		q.Write(i)
	}
	if q.Len() != total {
		t.Fatalf("len: expected %d, got %d", total, q.Len())
	}
	for i := 0; i < total; i++ {
		v, _ := q.Read()
		if v != i {
			t.Fatalf("pos %d: expected %d, got %d", i, i, v)
		}
	}
}

func TestSliceQueue_StringType(t *testing.T) {
	var q SliceQueue[string]
	q.Write("hello")
	q.Write("world")
	v, _ := q.Read()
	if v != "hello" {
		t.Fatalf("expected hello, got %s", v)
	}
	v, _ = q.Read()
	if v != "world" {
		t.Fatalf("expected world, got %s", v)
	}
}

func TestSliceQueueSafe_Concurrent(t *testing.T) {
	q := NewSliceQueueSafe[int]()
	var wg sync.WaitGroup
	const n = 10000

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			q.Write(i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			for {
				if _, err := q.Read(); err == nil {
					break
				}
			}
		}
	}()
	wg.Wait()
	if q.Len() != 0 {
		t.Fatalf("final len: expected 0, got %d", q.Len())
	}
}

func TestSliceQueueSafe_Reset(t *testing.T) {
	q := NewSliceQueueSafe[int]()
	q.Write(1)
	q.Reset()
	if q.Len() != 0 {
		t.Fatalf("len after reset: expected 0, got %d", q.Len())
	}
}
