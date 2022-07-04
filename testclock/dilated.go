// Copyright 2022 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package testclock

import (
	"time"

	"github.com/juju/clock"
)

// NewDilatedWallClock returns a clock that can be sped up or slowed down.
// realSecondDuration is the real duration of a second.
func NewDilatedWallClock(realSecondDuration time.Duration) clock.Clock {
	return &dilationClock{
		epoch:              time.Now(),
		realSecondDuration: realSecondDuration,
	}
}

type dilationClock struct {
	epoch              time.Time
	realSecondDuration time.Duration
}

// Now is part of the Clock interface.
func (dc *dilationClock) Now() time.Time {
	now := time.Now()
	return dc.epoch.Add(time.Duration(float64(now.Sub(dc.epoch)) / dc.realSecondDuration.Seconds()))
}

// After implements Clock.After.
func (dc *dilationClock) After(d time.Duration) <-chan time.Time {
	return time.After(time.Duration(float64(d) * dc.realSecondDuration.Seconds()))
}

// AfterFunc implements Clock.AfterFunc.
func (dc *dilationClock) AfterFunc(d time.Duration, f func()) clock.Timer {
	return dilatedWallTimer{
		timer:              time.AfterFunc(time.Duration(float64(d)*dc.realSecondDuration.Seconds()), f),
		realSecondDuration: dc.realSecondDuration,
	}
}

// NewTimer implements Clock.NewTimer.
func (dc *dilationClock) NewTimer(d time.Duration) clock.Timer {
	return dilatedWallTimer{
		timer:              time.NewTimer(time.Duration(float64(d) * dc.realSecondDuration.Seconds())),
		realSecondDuration: dc.realSecondDuration,
	}
}

// dilatedWallTimer implements the Timer interface.
type dilatedWallTimer struct {
	timer              *time.Timer
	realSecondDuration time.Duration
}

// Chan implements Timer.Chan.
func (t dilatedWallTimer) Chan() <-chan time.Time {
	return t.timer.C
}

func (t dilatedWallTimer) Reset(d time.Duration) bool {
	return t.timer.Reset(time.Duration(float64(d) * t.realSecondDuration.Seconds()))
}

func (t dilatedWallTimer) Stop() bool {
	return t.timer.Stop()
}
