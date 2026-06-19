// Package clock implements a Hybrid Logical Clock (HCL) for causality-preserving
// timestamps in the ANS receipt chain.
//
// HCL combines physical wall-clock time with a monotonic counter to ensure:
//   - Monotonicity: timestamps never go backwards, even if the wall clock is
//     adjusted or a previous timestamp was generated with a later clock value.
//   - Causality: if event A happens-before event B, then A.ts < B.ts
//     (assuming the same HLC instance).
//   - Physical correspondence: the timestamp approximates wall-clock time in
//     nanoseconds since Unix epoch, so existing display code using
//     time.Unix(0, ts) continues to work.
//
// The implementation tracks the maximum wall-clock nanosecond seen. If the
// wall clock advances, that value is used. If it stays the same or regresses,
// the previous maximum is incremented by 1. This guarantees strict
// monotonicity without a separate logical-counter encoding.
//
// SPDX-License-Identifier: MIT
package clock

import (
	"sync"
	"time"
)

// HLC is a Hybrid Logical Clock. Zero value is ready to use.
type HLC struct {
	mu       sync.Mutex
	physical int64 // max wall-clock UnixNano seen
}

// Now returns a monotonic timestamp in nanoseconds since Unix epoch.
// It is guaranteed to be strictly greater than any previous timestamp
// from this HLC instance.
func (h *HLC) Now() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()

	wall := time.Now().UnixNano()
	if wall > h.physical {
		h.physical = wall
	} else {
		h.physical++
	}
	return h.physical
}

// Physical is a no-op: the HLC value is already nanoseconds since epoch.
func Physical(ts int64) time.Time {
	return time.Unix(0, ts)
}
