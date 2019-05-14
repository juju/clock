// Copyright 2017 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package monotonic_test

import (
	"testing"
	"time"

	"github.com/juju/clock/monotonic"
)

func TestNow(t *testing.T) {
	var prev time.Duration
	for i := 0; i < 1000; i++ {
		val := monotonic.Now()
		if val < prev {
			t.Fatal("now is less than previous value")
		}
		prev = val
	}
}
