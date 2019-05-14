// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package testclock_test

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"

	"github.com/juju/clock/testclock"
)

func TestNow(t *testing.T) {
	c := qt.New(t)
	t0 := time.Now()
	cl := testclock.NewClock(t0)
	c.Assert(cl.Now(), qt.Equals, t0)
}

var (
	shortWait = 50 * time.Millisecond
	longWait  = time.Second
)

func TestAdvanceLogs(t *testing.T) {
	c := qt.New(t)
	t0 := time.Now()
	cl := testclock.NewClock(t0)
	var logBuf bytes.Buffer
	cl.SetLog(func(msg string) {
		fmt.Fprintln(&logBuf, msg)
	})

	// Shouldn't log anything.
	tc := cl.After(time.Second)
	cl.Advance(time.Minute)
	<-tc
	c.Check(logBuf.Bytes(), qt.HasLen, 0)

	// Should log since nothing's waiting.
	cl.Advance(time.Hour)
	c.Check(logBuf.String(), qt.Equals, "advancing a clock that has nothing waiting: cf. https://github.com/juju/juju/wiki/Intermittent-failures\n")
}

func TestWaitAdvance(t *testing.T) {
	c := qt.New(t)
	t0 := time.Now()
	cl := testclock.NewClock(t0)

	// It is legal to just say 'nothing is waiting'
	err := cl.WaitAdvance(0, 0, 0)
	c.Check(err, qt.Equals, nil)

	// Test that no timers errors out.
	err = cl.WaitAdvance(time.Millisecond, 10*time.Millisecond, 1)
	c.Check(err, qt.ErrorMatches, "got 0 timers added after waiting 10ms: wanted 1, stacks:\n")

	// Test that a timer doesn't error.
	_ = cl.After(time.Nanosecond)
	err = cl.WaitAdvance(time.Millisecond, 10*time.Millisecond, 1)
	c.Check(err, qt.Equals, nil)
}

func TestAdvanceWithAfter(t *testing.T) {
	c := qt.New(t)
	t0 := time.Now()
	cl := testclock.NewClock(t0)
	ch := cl.After(time.Second)
	select {
	case <-ch:
		c.Fatalf("received unexpected event")
	case <-time.After(shortWait):
	}

	cl.Advance(time.Second - 1)

	select {
	case <-ch:
		c.Fatalf("received unexpected event")
	case <-time.After(shortWait):
	}

	cl.Advance(1)

	select {
	case <-ch:
	case <-time.After(longWait):
		c.Fatalf("expected event to be triggered")
	}

	cl.Advance(time.Second)
	select {
	case <-ch:
		c.Fatalf("received unexpected event")
	case <-time.After(shortWait):
	}

	// Test that we can do it again
	ch = cl.After(time.Second)
	cl.Advance(2 * time.Second)
	select {
	case <-ch:
	case <-time.After(longWait):
		c.Fatalf("expected event to be triggered")
	}
	c.Assert(cl.Now().UTC(), qt.Equals, t0.Add(4*time.Second).UTC())
}

func TestAdvanceWithAfterFunc(t *testing.T) {
	c := qt.New(t)
	// Most of the details have been checked in TestAdvanceWithAfter,
	// so just check that AfterFunc is wired up correctly.
	t0 := time.Now()
	cl := testclock.NewClock(t0)
	fired := make(chan struct{})
	cl.AfterFunc(time.Second, func() {
		close(fired)
	})
	cl.Advance(2 * time.Second)
	select {
	case <-fired:
	case <-time.After(longWait):
		c.Fatalf("expected event to be triggered")
	}
}

func TestAfterFuncStop(t *testing.T) {
	c := qt.New(t)
	t0 := time.Now()
	cl := testclock.NewClock(t0)
	fired := make(chan struct{})
	timer := cl.AfterFunc(time.Second, func() {
		close(fired)
	})
	cl.Advance(50 * time.Millisecond)
	timer.Stop()
	select {
	case <-fired:
		c.Fatalf("received unexpected event")
	case <-time.After(shortWait):
	}
}

func TestNewTimerReset(t *testing.T) {
	c := qt.New(t)
	t0 := time.Now()
	cl := testclock.NewClock(t0)
	timer := cl.NewTimer(time.Second)
	cl.Advance(time.Second)
	select {
	case t := <-timer.Chan():
		c.Assert(t.UTC(), qt.Equals, t0.Add(time.Second).UTC())
	case <-time.After(longWait):
		c.Fatalf("expected event to be triggered")
	}

	timer.Reset(50 * time.Millisecond)
	cl.Advance(100 * time.Millisecond)
	select {
	case t := <-timer.Chan():
		c.Assert(t.UTC(), qt.Equals, t0.Add(time.Second+100*time.Millisecond).UTC())
	case <-time.After(longWait):
		c.Fatalf("expected event to be triggered")
	}
}

func TestNewTimerAsyncReset(t *testing.T) {
	c := qt.New(t)
	t0 := time.Now()
	clock := testclock.NewClock(t0)
	timer := clock.NewTimer(time.Hour)
	stop := make(chan struct{})
	stopped := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-stop:
		case t := <-timer.Chan():
			c.Errorf("timer accidentally ticked at: %v", t)
		case <-time.After(longWait):
			c.Errorf("test took too long")
		}
		close(stopped)
	}()
	// Just our goroutine, but we don't go so far as to trigger the wakeup.
	clock.WaitAdvance(1*time.Minute, 10*time.Millisecond, 1)
	// Reset shouldn't trigger a wakeup, just move when it thinks it will wake up.
	timer.Reset(time.Hour)
	clock.WaitAdvance(1*time.Minute, 10*time.Millisecond, 1)
	timer.Reset(time.Minute)
	clock.WaitAdvance(30*time.Second, 10*time.Millisecond, 1)
	// Now tell the goroutine to stop and start another one that *does* want to
	// wake up
	close(stop)
	select {
	case <-stopped:
	case <-time.After(longWait):
		c.Errorf("goroutine failed to stop")
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case t := <-timer.Chan():
			c.Logf("timer successfully ticked: %v", t)
		case <-time.After(longWait):
			c.Errorf("timer took too long")
		}
	}()
	// And advance the clock long enough to cause it to notice
	clock.WaitAdvance(30*time.Second, 10*time.Millisecond, 1)
	wg.Wait()
}

func TestNewTimerResetCausesWakeup(t *testing.T) {
	c := qt.New(t)
	t0 := time.Now()
	clock := testclock.NewClock(t0)
	timer1 := clock.NewTimer(time.Hour)
	timer2 := clock.NewTimer(time.Hour)
	timer3 := clock.NewTimer(time.Hour)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case t := <-timer1.Chan():
			c.Check(t0, qt.Equals, t)
		case <-time.After(longWait):
			c.Errorf("timer1 took too long to wake up")
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-timer2.Chan():
			c.Errorf("timer2 should not wake up")
		case <-time.After(shortWait):
			c.Logf("timer2 succesfully slept for 50ms")
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case t := <-timer3.Chan():
			// Even though the reset was negative, it triggers at 'now'
			c.Check(t0, qt.Equals, t)
		case <-time.After(longWait):
			c.Errorf("timer3 took too long to wake up")
		}
	}()
	// Reseting the timer to a time <= 0 should cause it to wake up on its
	// own, without needing an Advance or WaitAdvance to be done
	timer1.Reset(0)
	timer3.Reset(-1 * time.Second)
	wg.Wait()
}

func TestMultipleWaiters(t *testing.T) {
	c := qt.New(t)
	var wg sync.WaitGroup
	t0 := time.Date(2000, 01, 01, 01, 0, 0, 0, time.UTC)
	cl := testclock.NewClock(t0)

	total := 0
	start := func(f func()) {
		total++
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
	}
	start(func() {
		<-cl.After(50 * time.Millisecond)
	})
	start(func() {
		ch := make(chan struct{})
		cl.AfterFunc(100*time.Millisecond, func() {
			close(ch)
		})
		<-ch
	})
	start(func() {
		timer := cl.NewTimer(150 * time.Millisecond)
		<-timer.Chan()
		timer.Reset(50 * time.Millisecond)
		<-timer.Chan()
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for all the alarms to be waited on.
	for i := 0; i < total; i++ {
		select {
		case <-cl.Alarms():
		case <-time.After(longWait):
			c.Fatalf("expected a notification on the alarms channel")
		}
	}
	select {
	case <-cl.Alarms():
		c.Fatalf("unexpected extra notification on alarms channel")
	case <-time.After(shortWait):
	}

	cl.Advance(150 * time.Millisecond)

	// Wait for the extra notification after reset.
	select {
	case <-cl.Alarms():
	case <-time.After(longWait):
		c.Fatalf("expected a notification on the alarms channel")
	}

	cl.Advance(50 * time.Millisecond)

	select {
	case <-done:
	case <-time.After(longWait):
		c.Fatalf("expected all waits to complete")
	}

}
