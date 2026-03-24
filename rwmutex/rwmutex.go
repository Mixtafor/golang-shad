//go:build !solution

package rwmutex

// A RWMutex is a reader/writer mutual exclusion lock.
// The lock can be held by an arbitrary number of readers or a single writer.
// The zero value for a RWMutex is an unlocked mutex.
//
// If a goroutine holds a RWMutex for reading and another goroutine might
// call Lock, no goroutine should expect to be able to acquire a read lock
// until the initial read lock is released. In particular, this prohibits
// recursive read locking. This is to ensure that the lock eventually becomes
// available; a blocked Lock call excludes new readers from acquiring the
// lock.
type Mutex struct {
	mut chan struct{}
}

func (m *Mutex) Lock() {
	m.mut <- struct{}{}
}

func (m *Mutex) Unlock() {
	<-m.mut
}

type RWMutex struct {
	mutRead  Mutex
	mutWrite Mutex
	cnt      int
}

// New creates *RWMutex.
func New() *RWMutex {
	rw := RWMutex{mutRead: Mutex{make(chan struct{}, 1)},
		mutWrite: Mutex{make(chan struct{}, 1)}}
	return &rw
}

// RLock locks rw for reading.
//
// It should not be used for recursive read locking; a blocked Lock
// call excludes new readers from acquiring the lock. See the
// documentation on the RWMutex type.
// func (rw *RWMutex) release() {
// 	<-rw.mutexed
// }

func (rw *RWMutex) RLock() {
	rw.mutRead.Lock()
	rw.cnt++
	if rw.cnt == 1 {
		rw.mutWrite.Lock()
	}
	rw.mutRead.Unlock()
}

// RUnlock undoes a single RLock call;
// it does not affect other simultaneous readers.
// It is a run-time error if rw is not locked for reading
// on entry to RUnlock.
func (rw *RWMutex) RUnlock() {
	rw.mutRead.Lock()
	rw.cnt--
	if rw.cnt == 0 {
		rw.mutWrite.Unlock()
	}
	rw.mutRead.Unlock()
}

// Lock locks rw for writing.
// If the lock is already locked for reading or writing,
// Lock blocks until the lock is available.
func (rw *RWMutex) Lock() {
	rw.mutWrite.Lock()
}

// Unlock unlocks rw for writing. It is a run-time error if rw is
// not locked for writing on entry to Unlock.
//
// As with Mutexes, a locked RWMutex is not associated with a particular
// goroutine. One goroutine may RLock (Lock) a RWMutex and then
// arrange for another goroutine to RUnlock (Unlock) it.
func (rw *RWMutex) Unlock() {
	rw.mutWrite.Unlock()
}
