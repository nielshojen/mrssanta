package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunJobsRunsEachJobUntilCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var first, second atomic.Int32

	done := make(chan struct{})
	go func() {
		runJobs(ctx, []job{
			{name: "first", interval: time.Millisecond, runOnce: func(context.Context) error {
				first.Add(1)
				return nil
			}},
			{name: "second", interval: time.Hour, runOnce: func(context.Context) error {
				second.Add(1)
				return errors.New("keeps going after a failure")
			}},
		})
		close(done)
	}()

	assert.Eventually(t, func() bool { return first.Load() >= 3 && second.Load() == 1 }, time.Second, time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runJobs didn't return after cancel")
	}
	assert.Equal(t, int32(1), second.Load(), "the hourly job ran once, straight away")
}
