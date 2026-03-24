//go:build !solution

package keylock

import (
	"sync"
)

type Waiter struct {
	neededCnt int
	//	keys      []string
	blocker    chan struct{}
	usingKeys  []string
	inStarting bool
}
type KeyLock struct {
	mu            sync.Mutex
	keysNeeded    map[string][]*Waiter
	keysFree      map[string][]*Waiter
	allLockedKeys map[string]struct{}
}

func New() *KeyLock {
	kl := &KeyLock{keysFree: make(map[string][]*Waiter), keysNeeded: make(map[string][]*Waiter),
		allLockedKeys: make(map[string]struct{})}
	return kl
}

func (l *KeyLock) lockKeys(keys []string) { //mutex must be locked
	for _, key := range keys {
		l.allLockedKeys[key] = struct{}{}
		arr, ok := l.keysFree[key]
		if ok {
			for _, k := range arr {
				k.neededCnt++
			}
			l.keysNeeded[key] = append(l.keysNeeded[key], arr...)
			delete(l.keysFree, key)
		}
	}

}

func (l *KeyLock) freeKeys(keys []string) { //mutex must be locked
	for _, key := range keys {
		delete(l.allLockedKeys, key)
		if arr, ok := l.keysNeeded[key]; ok {
			for _, k := range arr {
				k.neededCnt--
			}
			l.keysFree[key] = append(l.keysFree[key], arr...)
			delete(l.keysNeeded, key)
		}
	}
}

func (l *KeyLock) removeWaiter(w *Waiter) { //mutex must be locked
	for _, storage := range [2]map[string][]*Waiter{l.keysNeeded, l.keysFree} {
		for _, key := range w.usingKeys {
			if _, ok := storage[key]; ok {
				arr := storage[key]
				for i, waiter := range arr {
					if waiter == w {
						arr[i] = arr[len(arr)-1]
						arr[len(arr)-1] = nil
						arr = arr[:len(arr)-1]
						break
					}
				}
				storage[key] = arr
			}
		}
	}
}

func (l *KeyLock) wakeUp(keys []string) { //mutex must be locked
	for _, key := range keys {
		if arr, ok := l.keysFree[key]; ok {
			for _, k := range arr {
				if k.neededCnt == 0 {
					k.inStarting = true
					l.lockKeys(k.usingKeys)
					l.removeWaiter(k)
					k.blocker <- struct{}{}
					break
				}
			}
		}
	}
}

func (l *KeyLock) LockKeys(keys []string, cancel <-chan struct{}) (canceled bool, unlock func()) {
	l.mu.Lock()

	neededCnt := 0
	curNeeded := make([]string, 0)
	curFree := make([]string, 0)
	for _, key := range keys {
		if _, ok := l.allLockedKeys[key]; ok {
			neededCnt++
			curNeeded = append(curNeeded, key)
		} else {
			curFree = append(curFree, key)
		}
	}

	waiter := &Waiter{blocker: make(chan struct{}, 1), neededCnt: neededCnt, usingKeys: keys}

	if neededCnt == 0 {
		l.lockKeys(keys)
		defer l.mu.Unlock()
		return false, func() {
			l.mu.Lock()
			defer l.mu.Unlock()
			l.freeKeys(keys)
			l.wakeUp(keys)
			waiter.inStarting = false
		}
	}

	for _, key := range curNeeded {
		l.keysNeeded[key] = append(l.keysNeeded[key], waiter)
	}
	for _, key := range curFree {
		l.keysFree[key] = append(l.keysFree[key], waiter)
	}

	l.mu.Unlock()

	select {
	case <-cancel:
		l.mu.Lock()
		defer l.mu.Unlock()

		if waiter.inStarting {
			l.freeKeys(keys)
			l.wakeUp(keys)
			waiter.inStarting = false
		} else {
			l.removeWaiter(waiter)
		}

		return true, func() {}

	case <-waiter.blocker:
		return false, func() {
			l.mu.Lock()
			defer l.mu.Unlock()
			l.freeKeys(keys)
			l.wakeUp(keys)
			waiter.inStarting = false
		}
	}
}
