// Copyright 2022 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package testclock_test

import (
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
