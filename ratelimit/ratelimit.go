//go:build !solution

package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Limiter is precise rate limiter with context support.
type Limiter struct {
	mutex    *sync.Mutex
	maxCount int
	interval time.Duration

	tokens chan struct{}

	taskCnt     int
	doneTaskCnt int
	stopped     bool
	queue       []*time.Timer
	once        *sync.Once

	stopBlocker1 chan struct{}
}

var ErrStopped = errors.New("limiter stopped")

// NewLimiter returns limiter that throttles rate of successful Acquire() calls
// to maxSize events at any given interval.
func NewLimiter(maxCount int, interval time.Duration) *Limiter {
	limiter := &Limiter{maxCount: maxCount, interval: interval,
		mutex: &sync.Mutex{}, tokens: make(chan struct{}, maxCount),
		stopBlocker1: make(chan struct{}), queue: make([]*time.Timer, 0),
		once: &sync.Once{}}

	for range maxCount {
		limiter.tokens <- struct{}{}
	}

	return limiter
}

func checkTasksCompl(l *Limiter) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.stopped && l.doneTaskCnt == l.taskCnt {
		for _, fun := range l.queue {
			fun.Stop()
		}
		l.once.Do(func() {
			close(l.stopBlocker1)
		})
		return
	}
	return
}

func (l *Limiter) Acquire(ctx context.Context) error {
	l.mutex.Lock()
	if l.stopped {
		l.mutex.Unlock()
		return ErrStopped
	}
	l.taskCnt++
	l.mutex.Unlock()

	select {
	case <-ctx.Done():
		l.mutex.Lock()
		l.doneTaskCnt++
		l.mutex.Unlock()
		checkTasksCompl(l)
		return ctx.Err()
	case <-l.tokens:
		l.mutex.Lock()
		l.queue = append(l.queue,
			time.AfterFunc(l.interval, func() {
				l.mutex.Lock()
				defer l.mutex.Unlock()
				l.queue[0] = nil
				l.queue = l.queue[1:]
				l.tokens <- struct{}{}
			}))
		l.mutex.Unlock()
	}

	l.mutex.Lock()
	l.doneTaskCnt++
	l.mutex.Unlock()

	checkTasksCompl(l)
	return nil
}

func (l *Limiter) Stop() {
	l.mutex.Lock()
	l.stopped = true
	l.mutex.Unlock()
	checkTasksCompl(l)
	<-l.stopBlocker1
}
