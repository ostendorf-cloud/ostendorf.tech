package feed

import (
	"context"
	"math/rand/v2"
	"time"
)

// retrySoonMax bounds how quickly the loop retries while not yet ready,
// mirroring old_simon-ostendorf.de's Syncer.Run, which retries sooner while
// the feed is still empty; here the trigger is broader (not ready at all,
// which also covers "OpenBao has no usable token yet"), but the shape —
// jittered interval, capped fast retry — is the same.
const retrySoonMax = 5 * time.Minute

// run is the background loop Start launches: sync, then wait, forever
// until ctx is done. A failed cycle is logged and never costs the last
// good feed, which Syncer.Sync already guarantees never gets replaced by
// nothing.
func (a *App) run(ctx context.Context) {
	for {
		if err := a.syncer.Sync(ctx); err != nil {
			if ctx.Err() == nil {
				a.logger.Warn("feed sync failed, keeping the last feed", "error", err.Error(), "posts", len(a.feed.Posts()))
			}
		} else {
			a.ready.Store(true)
		}

		wait := a.interval
		if !a.ready.Load() {
			wait = min(a.interval, retrySoonMax)
		}
		if !sleep(ctx, jitter(wait)) {
			return
		}
	}
}

// jitter keeps a restarted pod from hitting Meta and OpenBao on the same
// schedule as the one it replaced.
func jitter(d time.Duration) time.Duration {
	return d + time.Duration(rand.Int64N(int64(d/10)+1))
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
