package main

import (
	"context"
	"sync"
	"time"
)

// job is a background task the worker runs on a fixed interval.
type job struct {
	name     string
	interval time.Duration
	runOnce  func(ctx context.Context) error
}

// runJobs runs each job straight away and then every interval, until ctx is
// cancelled. A failed run is logged and retried on the next interval.
func runJobs(ctx context.Context, jobs []job) {
	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runEvery(ctx, j)
		}()
	}
	wg.Wait()
}

func runEvery(ctx context.Context, j job) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		if err := j.runOnce(ctx); err != nil {
			Errorf("%s failed: %v", j.name, err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
