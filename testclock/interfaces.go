package testclock

import (
	"time"

	"github.com/juju/clock"
)

// AdvanceableClock is a clock that can be advanced to trigger timers/trigger timers earlier
// than they would otherwise.
type AdvanceableClock interface {
	clock.Clock
	Advance(time.Duration)
}
