// Package tempo provides an internal clock, timeline, and beat/bar grid.
//
// Tempo is optional: an app that does not need timing can ignore this
// package entirely. A Clock tracks elapsed musical time from an explicit
// Start/Stop/Continue and exposes its position on a beat/bar grid. The
// package is audio- and MIDI-free; adapters that slave it to external
// clocks (MIDI clock, Ableton Link) live elsewhere.
package tempo

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// Position is a point on the beat/bar grid. Bar and Beat are zero-based;
// Phase is the fractional progress through the current beat, in [0,1).
type Position struct {
	Bar   int
	Beat  int
	Phase float64
}

// String renders the position in a compact, human-readable form.
func (p Position) String() string {
	return fmt.Sprintf("bar=%d beat=%d phase=%.2f", p.Bar+1, p.Beat+1, p.Phase)
}

// Clock is an internal musical clock. It measures wall-clock time from the
// moment Start (or Continue) is called and converts it to beats using the
// configured BPM. It is safe for concurrent use.
type Clock struct {
	mu           sync.RWMutex
	bpm          float64
	beatsPerBar  int
	running      bool
	segmentStart time.Time
	accumulated  time.Duration
}

// New returns a stopped clock at the given BPM. A non-positive BPM defaults
// to 120.
func New(bpm float64) *Clock {
	c := &Clock{beatsPerBar: 4}
	if bpm <= 0 {
		bpm = 120
	}
	c.bpm = bpm
	return c
}

// BPM returns the current tempo.
func (c *Clock) BPM() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.bpm
}

// SetBPM changes the tempo. A non-positive value is ignored.
func (c *Clock) SetBPM(bpm float64) {
	if bpm <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bpm = bpm
}

// BeatsPerBar returns the number of beats in one bar.
func (c *Clock) BeatsPerBar() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.beatsPerBar
}

// SetBeatsPerBar changes the meter. Values below 1 are ignored.
func (c *Clock) SetBeatsPerBar(n int) {
	if n < 1 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.beatsPerBar = n
}

// Start resets elapsed time to zero and starts the clock. Calling Start on
// an already running clock is a no-op.
func (c *Clock) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return
	}
	c.accumulated = 0
	c.segmentStart = time.Now()
	c.running = true
}

// Stop pauses the clock, retaining elapsed time. Calling Stop on a stopped
// clock is a no-op.
func (c *Clock) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return
	}
	c.accumulated += time.Since(c.segmentStart)
	c.running = false
}

// Continue resumes a stopped clock without resetting elapsed time. Calling
// Continue on a running clock is a no-op.
func (c *Clock) Continue() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return
	}
	c.segmentStart = time.Now()
	c.running = true
}

// Reset zeroes elapsed time. A running clock keeps running from now.
func (c *Clock) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accumulated = 0
	if c.running {
		c.segmentStart = time.Now()
	}
}

// Running reports whether the clock is currently advancing.
func (c *Clock) Running() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// Elapsed returns the musical time elapsed since Start (or the last Reset),
// excluding time spent stopped.
func (c *Clock) Elapsed() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.running {
		return c.accumulated + time.Since(c.segmentStart)
	}
	return c.accumulated
}

// Beats returns the total number of beats elapsed, including the fractional
// part of the current beat.
func (c *Clock) Beats() float64 {
	return c.Elapsed().Seconds() * c.BPM() / 60.0
}

// Position returns the current bar, beat, and phase.
func (c *Clock) Position() Position {
	total := c.Beats()
	whole := math.Floor(total)
	bpb := c.BeatsPerBar()
	if bpb < 1 {
		bpb = 1
	}
	n := int(whole)
	return Position{
		Bar:   n / bpb,
		Beat:  n % bpb,
		Phase: total - whole,
	}
}

// BeatDuration returns the duration of one beat at bpm.
func BeatDuration(bpm float64) time.Duration {
	if bpm <= 0 {
		bpm = 120
	}
	return time.Duration(60.0 / bpm * float64(time.Second))
}

// BarDuration returns the duration of one bar.
func BarDuration(bpm float64, beatsPerBar int) time.Duration {
	if beatsPerBar < 1 {
		beatsPerBar = 1
	}
	return time.Duration(beatsPerBar) * BeatDuration(bpm)
}

// SnapDownToBeat rounds d down to the previous beat boundary.
func SnapDownToBeat(d time.Duration, bpm float64) time.Duration {
	bd := BeatDuration(bpm)
	if bd <= 0 {
		return d
	}
	return d / bd * bd
}

// SnapDownToBar rounds d down to the previous bar boundary.
func SnapDownToBar(d time.Duration, bpm float64, beatsPerBar int) time.Duration {
	bd := BarDuration(bpm, beatsPerBar)
	if bd <= 0 {
		return d
	}
	return d / bd * bd
}

// SnapToNearestBeat rounds d to the nearest beat boundary.
func SnapToNearestBeat(d time.Duration, bpm float64) time.Duration {
	bd := BeatDuration(bpm)
	if bd <= 0 {
		return d
	}
	return time.Duration(math.Round(float64(d)/float64(bd))) * bd
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var digits []byte
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}
