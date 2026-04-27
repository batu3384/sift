// Package tui provides the terminal user interface for SIFT.
package tui

import (
	"sync"
	"time"
)

// debouncer limits the rate of function calls.
type debouncer struct {
	mu       sync.Mutex
	timer    *time.Timer
	delay    time.Duration
	pending  bool
}

// newDebouncer creates a debouncer with the specified delay.
func newDebouncer(delay time.Duration) *debouncer {
	return &debouncer{
		delay: delay,
	}
}

// Trigger schedules a callback to be called after the delay.
// If called again before the delay expires, the timer is reset.
func (d *debouncer) Trigger(callback func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		d.pending = false
		d.mu.Unlock()
		callback()
	})
	d.pending = true
}

// Cancel cancels any pending callback.
func (d *debouncer) Cancel() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	d.pending = false
}

// IsPending returns true if a callback is scheduled.
func (d *debouncer) IsPending() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pending
}

// CursorDebouncer is a debouncer specifically for cursor movement events.
type CursorDebouncer struct {
	*debouncer
	lastCursor int
}

// NewCursorDebouncer creates a debouncer optimized for cursor movement.
// The default delay is 150ms which provides responsive feel while preventing excessive preview loads.
func NewCursorDebouncer() *CursorDebouncer {
	return &CursorDebouncer{
		debouncer: newDebouncer(150 * time.Millisecond),
	}
}

// OnCursorChange triggers the debounced callback only if cursor actually changed.
func (d *CursorDebouncer) OnCursorChange(cursor int, callback func()) {
	if cursor == d.lastCursor {
		return
	}
	d.lastCursor = cursor
	d.Trigger(callback)
}

// ResetCursor resets the last cursor value (e.g., when leaving a screen).
func (d *CursorDebouncer) ResetCursor() {
	d.mu.Lock()
	d.lastCursor = -1
	d.mu.Unlock()
}

// PreviewDebouncer is a debouncer for preview loading with longer delay.
type PreviewDebouncer struct {
	*debouncer
}

// NewPreviewDebouncer creates a debouncer optimized for preview loading.
// The default delay is 200ms which balances responsiveness with avoiding redundant loads.
func NewPreviewDebouncer() *PreviewDebouncer {
	return &PreviewDebouncer{
		debouncer: newDebouncer(200 * time.Millisecond),
	}
}

// SearchDebouncer is a debouncer for search input with shorter delay.
type SearchDebouncer struct {
	*debouncer
}

// NewSearchDebouncer creates a debouncer optimized for search input.
// The default delay is 100ms for responsive search-as-you-type.
func NewSearchDebouncer() *SearchDebouncer {
	return &SearchDebouncer{
		debouncer: newDebouncer(100 * time.Millisecond),
	}
}