//go:build !windows

package lock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

// retrySleep is the interval between non-blocking flock retries in the
// blocking File path. Small enough that contention resolution feels
// snappy; large enough that we don't burn a CPU spinning.
const retrySleep = 100 * time.Millisecond

func lockFile(ctx context.Context, f *os.File) (func(), error) {
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("flock: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("acquire lock: %w", ctx.Err())
		case <-time.After(retrySleep):
		}
	}
}

func tryLockFile(f *os.File) (func(), error) {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrContended
		}
		return nil, fmt.Errorf("flock: %w", err)
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }, nil
}
