//go:build !solution

package cond

// A Locker represents an object that can be locked and unlocked.
type Locker interface {
	Lock()
	Unlock()
}

// Cond implements a condition variable, a rendezvous point
// for goroutines waiting for or announcing the occurrence
// of an event.
//
// Each Cond has an associated Locker L (often a *sync.Mutex or *sync.RWMutex),
// which must be held when changing the condition and
// when calling the Wait method.
type Cond struct {
	L       Locker
	queue   []chan struct{}
	mutexed chan struct{}
}

func (c *Cond) release() {
	<-c.mutexed
}

// New returns a new Cond with Locker l.
func New(l Locker) *Cond {
	return &Cond{L: l, mutexed: make(chan struct{}, 1)}
}

// Wait atomically unlocks c.L and suspends execution
// of the calling goroutine. After later resuming execution,
// Wait locks c.L before returning. Unlike in other systems,
// Wait cannot return unless awoken by Broadcast or Signal.
//
// Because c.L is not locked when Wait first resumes, the caller
// typically cannot assume that the condition is true when
// Wait returns. Instead, the caller should Wait in a loop:
//
//	c.L.Lock()
//	for !condition() {
//	    c.Wait()
//	}
//	... make use of condition ...
//	c.L.Unlock()
func (c *Cond) Wait() {
	ch := make(chan struct{})

	c.mutexed <- struct{}{}
	c.queue = append(c.queue, ch)
	c.release()
	c.L.Unlock()

	<-ch
	c.L.Lock()

}

// Signal wakes one goroutine waiting on c, if there is any.
//
// It is allowed but not required for the caller to hold c.L
// during the call.
func (c *Cond) Signal() {
	defer c.release()
	c.mutexed <- struct{}{}
	if len(c.queue) == 0 {
		return
	}

	frontCh := c.queue[0]
	c.queue[0] = nil
	c.queue = c.queue[1:]
	close(frontCh)
}

// Broadcast wakes all goroutines waiting on c.
//
// It is allowed but not required for the caller to hold c.L
// during the call.
func (c *Cond) Broadcast() {
	defer c.release()
	c.mutexed <- struct{}{}
	if len(c.queue) == 0 {
		return
	}
	for _, ch := range c.queue {
		close(ch)
	}
	c.queue = c.queue[:0]
}
