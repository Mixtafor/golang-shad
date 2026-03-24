//go:build !solution

package lrucache

import (
	"container/list"
	"sync"
)

type pair struct {
	value   int
	nodePtr *list.Element
}

type LRUstorage struct {
	_        sync.Locker // placeholder no copy
	cap      int
	queue    list.List // keys
	elements map[int]pair
}

func (l *LRUstorage) Get(key int) (int, bool) {
	elem, ok := l.elements[key]
	if !ok {
		return 0,
			false
	}

	l.queue.MoveToBack(elem.nodePtr)
	return elem.value, true
}

func (l *LRUstorage) Set(key, value int) {
	if l.cap == 0 {
		return
	}

	if elem, ok := l.elements[key]; ok {
		l.queue.MoveToBack(elem.nodePtr)
		elem.value = value
		l.elements[key] = elem
		return
	}

	if l.cap <= l.queue.Len() {
		frontEl := l.queue.Front()

		typedVal, ok := frontEl.Value.(int)
		if !ok {
			panic("in queue not int")
		}

		delete(l.elements, typedVal)
		l.queue.Remove(frontEl)
	}

	lastPtr := l.queue.PushBack(key)
	l.elements[key] = pair{value: value, nodePtr: lastPtr}
}

func (l *LRUstorage) Clear() {
	for k := range l.elements {
		delete(l.elements, k)
	}

	if l.queue.Len() > 0 {
		prev := l.queue.Front()
		next := prev.Next()
		for prev != nil {
			l.queue.Remove(prev)
			prev = next
			if next != nil {
				next = next.Next()
			}
		}
	}
}

func (l *LRUstorage) Range(f func(key, value int) bool) {
	for head := l.queue.Front(); head != nil; head = head.Next() {
		typedHeadVal, ok := head.Value.(int)
		if !ok {
			panic("in queue not int")
		}
		retVal := f(typedHeadVal, l.elements[typedHeadVal].value)
		if !retVal {
			break
		}
	}
}

func New(cap int) Cache {
	stor := &LRUstorage{cap: cap, queue: list.List{}, elements: make(map[int]pair)}
	stor.queue.Init()
	return stor
}
