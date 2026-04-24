//go:build !solution

package batcher

import (
	"sync"

	"gitlab.com/slon/shad-go/batcher/slow"
)

type Compute struct {
}

type Result struct {
	res any
}

type Batcher struct {
	v      *slow.Value
	o      *sync.Once
	mu     *sync.RWMutex
	res    *Result // ptr to res
	ready  bool
	isComp *bool
	isEnd  *bool
	mucomp *sync.RWMutex
	cond   *sync.Cond
	wg     *sync.WaitGroup
	wgres  *sync.WaitGroup
	cmp    *Compute
	cnt    *int
}

func (b *Batcher) Load() any {
	b.mucomp.Lock()
	locwg := b.wg
	locres := b.res
	locwgres := b.wgres
	loconce := b.o
	loccnt := b.cnt
	lociscomp := b.isComp
	locisend := b.isEnd
	(*loccnt)++
	b.wg.Add(1)

	if *lociscomp {
		b.wg.Done()
		b.mucomp.Unlock()
		locwg.Wait()
	}

	if !*locisend {
		*lociscomp = true
		b.mucomp.Unlock()
		val := b.v.Load()
		b.mucomp.Lock()
		locwgres.Add(1)

		*locisend = true

		if *loccnt == 1 {
			b.wg = new(sync.WaitGroup)
			b.res = new(Result)
			b.wgres = new(sync.WaitGroup)
			b.o = new(sync.Once)
			*b.cnt = 0
			*locisend = false
			*lociscomp = false
			defer b.mucomp.Unlock()
		}
		locwg.Done()
		return val
	}

	loconce.Do(func() {
		b.wg = new(sync.WaitGroup)
		b.wg.Add(1)
		b.res = new(Result)
		b.wgres = new(sync.WaitGroup)
		b.wgres.Add(1)
		b.o = new(sync.Once)
		var zero int
		b.cnt = &zero
		isComp := true
		isEnd := true
		b.isComp = &isComp
		b.isEnd = &isEnd
		b.mucomp.Unlock()
		*locres = Result{res: b.v.Load()}
		b.mucomp.Lock()
		locwgres.Done()

		if *b.cnt == 0 {
			b.wg = new(sync.WaitGroup)
			b.res = new(Result)
			b.wgres = new(sync.WaitGroup)
			b.o = new(sync.Once)
			*b.cnt = 0

			*b.isEnd = false
			*b.isComp = false
			defer b.mucomp.Unlock()
		} else {
			b.wg.Done()
		}

	})
	locwgres.Wait()
	return locres.res
}

func NewBatcher(v *slow.Value) *Batcher {
	var zero int
	var zeroIsComp bool
	var zeroIsEnd bool
	b := &Batcher{v: v,
		mu:     &sync.RWMutex{},
		mucomp: &sync.RWMutex{},
		wg:     &sync.WaitGroup{},
		wgres:  &sync.WaitGroup{},
		res:    new(Result),
		o:      new(sync.Once),
		cnt:    &zero,
		isComp: &zeroIsComp,
		isEnd:  &zeroIsEnd,
	}

	b.cond = sync.NewCond(b.mucomp)
	return b
}

