package pipeline

import (
	"context"
	"log"
	"os"
	"time"
)

var warnLog = log.New(os.Stderr, "warn: ", 0)

func warnf(format string, args ...any) { warnLog.Printf(format, args...) }

var retryWaits = []time.Duration{0, 2 * time.Second, 8 * time.Second}

type attempt[T any] struct {
	name string
	fn   func() (T, error)
}

func tryInOrder[T any](ctx context.Context, label string, attempts []attempt[T]) (T, error) {
	var zero T
	var last error
	for i, a := range attempts {
		wait := time.Duration(0)
		if i < len(retryWaits) {
			wait = retryWaits[i]
		}
		if wait > 0 {
			t := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				t.Stop()
				return zero, ctx.Err()
			case <-t.C:
			}
		}
		v, err := a.fn()
		if err == nil {
			if i > 0 {
				warnf("%s: succeeded with %q on attempt %d of %d after earlier failures", label, a.name, i+1, len(attempts))
			}
			return v, nil
		}
		warnf("%s: attempt %d of %d with %q failed: %v", label, i+1, len(attempts), a.name, err)
		last = err
	}
	return zero, last
}
