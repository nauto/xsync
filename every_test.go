package xsync

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAtMostRunsAtLeastOnce(t *testing.T) {
	assert := assert.New(t)

	atMostOncePerHour := AtMost{Every: time.Hour}
	numRuns := int32(0)

	logExecution := func() {
		atomic.AddInt32(&numRuns, 1)
	}

	// Run the first time.
	atMostOncePerHour.Run(logExecution)

	// Try running several more times.
	go func() {
		for i := 0; i != 100; i++ {
			atMostOncePerHour.Run(logExecution)
		}
	}()

	// Wait a bit to see if anything else has run.
	time.Sleep(time.Second)

	assert.EqualValues(atomic.LoadInt32(&numRuns), 1)
}

func TestAtMostRunsMoreThanOnce(t *testing.T) {
	assert := assert.New(t)

	atMostOncePerNs := AtMost{Every: time.Nanosecond}
	numRuns := int32(0)

	logExecution := func() {
		atomic.AddInt32(&numRuns, 1)
	}

	// Run twice.
	go func() {
		atMostOncePerNs.Run(logExecution)
		atMostOncePerNs.Run(logExecution)
	}()

	// Wait a bit to see if both runs happened.
	time.Sleep(time.Second)

	assert.EqualValues(atomic.LoadInt32(&numRuns), 2)
}
