package tempo

import (
	"testing"
	"time"
)

func TestNewDefaults(t *testing.T) {
	c := New(0)
	if c.BPM() != 120 {
		t.Fatalf("BPM = %v, want 120", c.BPM())
	}
	if c.BeatsPerBar() != 4 {
		t.Fatalf("BeatsPerBar = %d, want 4", c.BeatsPerBar())
	}
	if c.Running() {
		t.Fatal("new clock should be stopped")
	}
}

func TestSettersIgnoreInvalid(t *testing.T) {
	c := New(100)
	c.SetBPM(0)
	if c.BPM() != 100 {
		t.Fatalf("BPM changed on invalid input")
	}
	c.SetBeatsPerBar(0)
	if c.BeatsPerBar() != 4 {
		t.Fatalf("BeatsPerBar changed on invalid input")
	}
	c.SetBPM(140)
	c.SetBeatsPerBar(3)
	if c.BPM() != 140 || c.BeatsPerBar() != 3 {
		t.Fatalf("valid setters did not apply")
	}
}

func TestStartStopContinue(t *testing.T) {
	c := New(120)
	c.Start()
	if !c.Running() {
		t.Fatal("Start did not run")
	}
	time.Sleep(20 * time.Millisecond)
	c.Stop()
	if c.Running() {
		t.Fatal("Stop did not stop")
	}
	elapsed := c.Elapsed()
	if elapsed < 10*time.Millisecond {
		t.Fatalf("elapsed too small: %v", elapsed)
	}
	time.Sleep(20 * time.Millisecond)
	if c.Elapsed() != elapsed {
		t.Fatal("stopped clock advanced")
	}
	c.Continue()
	if !c.Running() {
		t.Fatal("Continue did not run")
	}
	time.Sleep(10 * time.Millisecond)
	if c.Elapsed() <= elapsed {
		t.Fatal("continued clock did not advance")
	}
}

func TestReset(t *testing.T) {
	c := New(120)
	c.Start()
	time.Sleep(20 * time.Millisecond)
	c.Reset()
	if c.Elapsed() > 5*time.Millisecond {
		t.Fatalf("Reset did not zero elapsed: %v", c.Elapsed())
	}
	if !c.Running() {
		t.Fatal("Reset stopped a running clock")
	}
}

func TestPosition(t *testing.T) {
	c := New(120) // 0.5s per beat, 2s per bar
	c.Start()
	time.Sleep(1100 * time.Millisecond)
	c.Stop()
	p := c.Position()
	// ~2.2 beats => bar 0, beat 2, phase ~0.2
	if p.Bar != 0 {
		t.Fatalf("Bar = %d, want 0", p.Bar)
	}
	if p.Beat != 2 {
		t.Fatalf("Beat = %d, want 2", p.Beat)
	}
	if p.Phase < 0.1 || p.Phase > 0.4 {
		t.Fatalf("Phase = %v, want ~0.2", p.Phase)
	}
}

func TestBeatAndBarDuration(t *testing.T) {
	if got := BeatDuration(120); got != 500*time.Millisecond {
		t.Fatalf("BeatDuration = %v", got)
	}
	if got := BarDuration(120, 4); got != 2*time.Second {
		t.Fatalf("BarDuration = %v", got)
	}
	if got := BarDuration(120, 0); got != 500*time.Millisecond {
		t.Fatalf("BarDuration invalid meter = %v", got)
	}
}

func TestSnap(t *testing.T) {
	bd := BeatDuration(120)
	if got := SnapDownToBeat(1100*time.Millisecond, 120); got != 2*bd {
		t.Fatalf("SnapDownToBeat = %v, want %v", got, 2*bd)
	}
	if got := SnapDownToBar(2500*time.Millisecond, 120, 4); got != 2*time.Second {
		t.Fatalf("SnapDownToBar = %v, want 2s", got)
	}
	if got := SnapToNearestBeat(450*time.Millisecond, 120); got != bd {
		t.Fatalf("SnapToNearestBeat = %v, want %v", got, bd)
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := New(120)
	c.Start()
	defer c.Stop()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			c.SetBPM(120 + float64(i%10))
			c.BPM()
			c.Position()
		}
		close(done)
	}()
	<-done
}
