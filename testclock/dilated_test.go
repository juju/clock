// Copyright 2022 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package testclock_test

import (
	"runtime"
	"sync"
	"time"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/clock/testclock"
)

type dilatedClockSuite struct {
	testing.LoggingSuite
}

var _ = gc.Suite(&dilatedClockSuite{})

func (*dilatedClockSuite) TestSlowedAfter(c *gc.C) {
	t0 := time.Now()
	cl := testclock.NewDilatedWallClock(2 * time.Second)
	t1 := <-cl.After(time.Second)
	c.Assert(t1.Sub(t0).Seconds(), jc.GreaterThan, 1.9)
}

func (*dilatedClockSuite) TestFastAfter(c *gc.C) {
	t0 := time.Now()
	cl := testclock.NewDilatedWallClock(500 * time.Millisecond)
	t1 := <-cl.After(time.Second)
	c.Assert(t1.Sub(t0).Milliseconds(), jc.LessThan, 600)
}

func (*dilatedClockSuite) TestSlowedAfterFunc(c *gc.C) {
	t0 := time.Now()
	cl := testclock.NewDilatedWallClock(2 * time.Second)
	mut := sync.Mutex{}
	mut.Lock()
	cl.AfterFunc(time.Second, func() {
		defer mut.Unlock()
		c.Check(time.Since(t0).Seconds(), jc.GreaterThan, 1.9)
	})
	mut.Lock()
}

func (*dilatedClockSuite) TestFastAfterFunc(c *gc.C) {
	t0 := time.Now()
	cl := testclock.NewDilatedWallClock(500 * time.Millisecond)
	mut := sync.Mutex{}
	mut.Lock()
	cl.AfterFunc(time.Second, func() {
		defer mut.Unlock()
		c.Check(time.Since(t0).Milliseconds(), jc.LessThan, 600)
	})
	mut.Lock()
}

func (*dilatedClockSuite) TestSlowedNow(c *gc.C) {
	t0 := time.Now()
	cl := testclock.NewDilatedWallClock(2 * time.Second)
	<-time.After(time.Second)
	t2 := cl.Now()
	c.Assert(t2.Sub(t0).Milliseconds(), jc.GreaterThan, 400)
	c.Assert(t2.Sub(t0).Milliseconds(), jc.LessThan, 600)
	<-time.After(time.Second)
	t3 := cl.Now()
	c.Assert(t3.Sub(t0).Milliseconds(), jc.GreaterThan, 900)
	c.Assert(t3.Sub(t0).Milliseconds(), jc.LessThan, 1100)
}

func (*dilatedClockSuite) TestFastNow(c *gc.C) {
	t0 := time.Now()
	cl := testclock.NewDilatedWallClock(500 * time.Millisecond)
	<-time.After(time.Second)
	t2 := cl.Now()
	c.Assert(t2.Sub(t0).Milliseconds(), jc.GreaterThan, 1900)
	c.Assert(t2.Sub(t0).Milliseconds(), jc.LessThan, 2100)
	<-time.After(time.Second)
	t3 := cl.Now()
	c.Assert(t3.Sub(t0).Milliseconds(), jc.GreaterThan, 3900)
	c.Assert(t3.Sub(t0).Milliseconds(), jc.LessThan, 4100)
}

func (*dilatedClockSuite) TestAdvance(c *gc.C) {
	t0 := time.Now()
	cl := testclock.NewDilatedWallClock(500 * time.Millisecond)
	first := cl.After(1 * time.Second)
	cl.Advance(500 * time.Millisecond)
	<-time.After(250 * time.Millisecond)
	select {
	case t := <-first:
		c.Assert(t.Sub(t0).Milliseconds(), jc.GreaterThan, 249)
	case <-time.After(50 * time.Millisecond):
		c.Fatal("timer failed to trigger early")
	}
}

func (*dilatedClockSuite) TestAdvanceMulti(c *gc.C) {
	cl := testclock.NewDilatedWallClock(500 * time.Millisecond)
	first := cl.After(1 * time.Second)
	second := cl.After(2 * time.Second)
	third := cl.After(1 * time.Hour)
	fourth := cl.After(24 * time.Hour)
	cl.Advance(12 * time.Hour)
	n := 0
	done := time.After(10 * time.Second)
out:
	for {
		select {
		case <-first:
			n++
		case <-second:
			n++
		case <-third:
			n++
		case <-fourth:
			c.Fatal("timer that fired that should not have")
		case <-done:
			break out
		}
	}
	c.Assert(n, gc.Equals, 3)
}

func (*dilatedClockSuite) TestStop(c *gc.C) {
	numGo := runtime.NumGoroutine()
	cl := testclock.NewDilatedWallClock(500 * time.Millisecond)
	a := cl.NewTimer(1 * time.Second)
	time.Sleep(100 * time.Millisecond)
	ok := a.Stop()
	c.Assert(ok, jc.IsTrue)
	ok = a.Stop()
	c.Assert(ok, jc.IsFalse)
	select {
	case <-a.Chan():
		c.Fatal("stopped clock fired")
	case <-time.After(1 * time.Second):
	}
	time.Sleep(50 * time.Millisecond)
	numGoAfter := runtime.NumGoroutine()
	c.Assert(numGoAfter, gc.Equals, numGo, gc.Commentf("clock goroutine still running"))
}

func (*dilatedClockSuite) TestReset(c *gc.C) {
	numGo := runtime.NumGoroutine()
	cl := testclock.NewDilatedWallClock(500 * time.Millisecond)
	a := cl.NewTimer(1 * time.Second)
	time.Sleep(250 * time.Millisecond)
	ok := a.Reset(1 * time.Second)
	c.Assert(ok, jc.IsTrue)
	<-time.After(500 * time.Millisecond)
	select {
	case <-a.Chan():
	case <-time.After(50 * time.Millisecond):
		c.Fatal("clock did not fire")
	}
	time.Sleep(50 * time.Millisecond)
	numGoAfter := runtime.NumGoroutine()
	c.Assert(numGoAfter, gc.Equals, numGo, gc.Commentf("clock goroutine still running"))
}

func (*dilatedClockSuite) TestStopReset(c *gc.C) {
	numGo := runtime.NumGoroutine()
	cl := testclock.NewDilatedWallClock(500 * time.Millisecond)
	a := cl.NewTimer(1 * time.Second)
	time.Sleep(250 * time.Millisecond)
	ok := a.Stop()
	c.Assert(ok, jc.IsTrue)
	ok = a.Reset(1 * time.Second)
	c.Assert(ok, jc.IsTrue)
	<-time.After(500 * time.Millisecond)
	select {
	case <-a.Chan():
	case <-time.After(50 * time.Millisecond):
		c.Fatal("clock did not fire")
	}
	time.Sleep(50 * time.Millisecond)
	numGoAfter := runtime.NumGoroutine()
	c.Assert(numGoAfter, gc.Equals, numGo, gc.Commentf("clock goroutine still running"))
}
