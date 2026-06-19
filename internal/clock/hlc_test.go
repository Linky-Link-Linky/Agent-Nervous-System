package clock

import (
	"sync"
	"testing"
	"time"
)

func TestHLCIncreases(t *testing.T) {
	var h HLC
	prev := h.Now()
	for i := 0; i < 1000; i++ {
		cur := h.Now()
		if cur <= prev {
			t.Fatalf("HLC did not increase: prev=%d, cur=%d", prev, cur)
		}
		prev = cur
	}
}

func TestHLCConcurrent(t *testing.T) {
	var h HLC
	var wg sync.WaitGroup
	n := 50
	ts := make([]int64, n)
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ts[idx] = h.Now()
		}(i)
	}
	wg.Wait()

	seen := make(map[int64]bool)
	for _, tsVal := range ts {
		if seen[tsVal] {
			t.Errorf("duplicate HLC timestamp: %d", tsVal)
		}
		seen[tsVal] = true
	}
}

func TestHLCClockRegression(t *testing.T) {
	var h HLC
	first := h.Now()
	second := h.Now()
	if second <= first {
		t.Errorf("HLC did not advance on second call: first=%d, second=%d", first, second)
	}
}

func TestPhysicalTime(t *testing.T) {
	var h HLC
	ts := h.Now()
	pt := Physical(ts)

	if pt.IsZero() {
		t.Error("Physical() returned zero time")
	}
	// Should be within a few seconds of now
	if diff := time.Since(pt); diff > 5*time.Second || diff < -5*time.Second {
		t.Errorf("Physical time off by %v", diff)
	}
}

func TestPhysicalMonotonic(t *testing.T) {
	var h HLC
	ts1 := h.Now()
	ts2 := h.Now()
	p1 := Physical(ts1)
	p2 := Physical(ts2)
	if p2.Before(p1) {
		t.Error("Physical time went backwards")
	}
}
