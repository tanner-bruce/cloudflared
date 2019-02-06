package origin

import (
	"context"
	"testing"
	"time"
)

func immediateTimeAfter(time.Duration) <-chan time.Time {
	c := make(chan time.Time, 1)
	c <- time.Now()
	return c
}

func TestBackoffRetries(t *testing.T) {
	// make backoff return immediately
	timeAfter = immediateTimeAfter
	ctx := context.Background()
	backoff := BackoffHandler{MaxRetries: 3}
	if !backoff.Backoff(ctx) {
		t.Fatalf("backoff failed immediately")
	}
	if !backoff.Backoff(ctx) {
		t.Fatalf("backoff failed after 1 retry")
	}
	if !backoff.Backoff(ctx) {
		t.Fatalf("backoff failed after 2 retry")
	}
	if backoff.Backoff(ctx) {
		t.Fatalf("backoff allowed after 3 (max) retries")
	}
}

func TestBackoffCancel(t *testing.T) {
	// prevent backoff from returning normally
	timeAfter = func(time.Duration) <-chan time.Time { return make(chan time.Time) }
	ctx, cancelFunc := context.WithCancel(context.Background())
	backoff := BackoffHandler{MaxRetries: 3}
	cancelFunc()
	if backoff.Backoff(ctx) {
		t.Fatalf("backoff allowed after cancel")
	}
	if _, ok := backoff.GetBackoffDuration(ctx); ok {
		t.Fatalf("backoff allowed after cancel")
	}
}

func TestBackoffGracePeriod(t *testing.T) {
	currentTime := time.Now()
	// make timeNow return whatever we like
	timeNow = func() time.Time { return currentTime }
	// make backoff return immediately
	timeAfter = immediateTimeAfter
	ctx := context.Background()
	backoff := BackoffHandler{MaxRetries: 1}
	if !backoff.Backoff(ctx) {
		t.Fatalf("backoff failed immediately")
	}
	// the next call to Backoff would fail unless it's after the grace period
	backoff.SetGracePeriod()
	// advance time to after the grace period (~4 seconds) and see what happens
	currentTime = currentTime.Add(time.Second * 5)
	if !backoff.Backoff(ctx) {
		t.Fatalf("backoff failed after the grace period expired")
	}
	// confirm we ignore grace period after backoff
	if backoff.Backoff(ctx) {
		t.Fatalf("backoff allowed after 1 (max) retry")
	}
}

func TestGetBackoffDurationRetries(t *testing.T) {
	// make backoff return immediately
	timeAfter = immediateTimeAfter
	ctx := context.Background()
	backoff := BackoffHandler{MaxRetries: 3}
	if _, ok := backoff.GetBackoffDuration(ctx); !ok {
		t.Fatalf("backoff failed immediately")
	}
	backoff.Backoff(ctx) // noop
	if _, ok := backoff.GetBackoffDuration(ctx); !ok {
		t.Fatalf("backoff failed after 1 retry")
	}
	backoff.Backoff(ctx) // noop
	if _, ok := backoff.GetBackoffDuration(ctx); !ok {
		t.Fatalf("backoff failed after 2 retry")
	}
	backoff.Backoff(ctx) // noop
	if _, ok := backoff.GetBackoffDuration(ctx); ok {
		t.Fatalf("backoff allowed after 3 (max) retries")
	}
	if backoff.Backoff(ctx) {
		t.Fatalf("backoff allowed after 3 (max) retries")
	}
}

func TestGetBackoffDuration(t *testing.T) {
	// make backoff return immediately
	timeAfter = immediateTimeAfter
	ctx := context.Background()
	backoff := BackoffHandler{MaxRetries: 3}
	if duration, ok := backoff.GetBackoffDuration(ctx); !ok || duration != time.Second {
		t.Fatalf("backoff didn't return 1 second on first retry")
	}
	backoff.Backoff(ctx) // noop
	if duration, ok := backoff.GetBackoffDuration(ctx); !ok || duration != time.Second*2 {
		t.Fatalf("backoff didn't return 2 seconds on second retry")
	}
	backoff.Backoff(ctx) // noop
	if duration, ok := backoff.GetBackoffDuration(ctx); !ok || duration != time.Second*4 {
		t.Fatalf("backoff didn't return 4 seconds on third retry")
	}
	backoff.Backoff(ctx) // noop
	if duration, ok := backoff.GetBackoffDuration(ctx); ok || duration != 0 {
		t.Fatalf("backoff didn't return 0 seconds on fourth retry (exceeding limit)")
	}
}

func TestBackoffRetryForever(t *testing.T) {
	// make backoff return immediately
	timeAfter = immediateTimeAfter
	ctx := context.Background()
	backoff := BackoffHandler{MaxRetries: 3, RetryForever: true}
	if duration, ok := backoff.GetBackoffDuration(ctx); !ok || duration != time.Second {
		t.Fatalf("backoff didn't return 1 second on first retry")
	}
	backoff.Backoff(ctx) // noop
	if duration, ok := backoff.GetBackoffDuration(ctx); !ok || duration != time.Second*2 {
		t.Fatalf("backoff didn't return 2 seconds on second retry")
	}
	backoff.Backoff(ctx) // noop
	if duration, ok := backoff.GetBackoffDuration(ctx); !ok || duration != time.Second*4 {
		t.Fatalf("backoff didn't return 4 seconds on third retry")
	}
	if !backoff.Backoff(ctx) {
		t.Fatalf("backoff refused on fourth retry despire RetryForever")
	}
	if duration, ok := backoff.GetBackoffDuration(ctx); !ok || duration != time.Second*8 {
		t.Fatalf("backoff returned %v instead of 8 seconds on fourth retry", duration)
	}
	if !backoff.Backoff(ctx) {
		t.Fatalf("backoff refused on fifth retry despire RetryForever")
	}
	if duration, ok := backoff.GetBackoffDuration(ctx); !ok || duration != time.Second*8 {
		t.Fatalf("backoff returned %v instead of 8 seconds on fifth retry", duration)
	}
}
