//go:build !solution

package pubsub

import (
	"context"
	"sync"
	"sync/atomic"
)

var _ Subscription = (*MySubscription)(nil)

type MySubscription struct {
	cb        MsgHandler
	subj      string
	ps        *MyPubSub
	isAlive   atomic.Bool
	wrkCtx    context.Context
	wrkCancel context.CancelFunc
	in        chan any
	out       chan any
}

func (s *MySubscription) Unsubscribe() {
	s.ps.mu.Lock()
	actQ := s.ps.subjs[s.subj]
	actQ.mu.Lock()

	for i, el := range actQ.subs {
		if el == s {
			actQ.subs[i].wrkCancel()
			close(actQ.subs[i].in)
			actQ.subs[i].isAlive.Store(false)
			actQ.subs[i] = actQ.subs[len(actQ.subs)-1]
			actQ.subs[len(actQ.subs)-1] = nil
			actQ.subs = actQ.subs[:len(actQ.subs)-1]
			break
		}
	}

	s.ps.subjs[s.subj] = actQ
	actQ.mu.Unlock()
	s.ps.mu.Unlock()
}

var _ PubSub = (*MyPubSub)(nil)

type queue struct {
	mu   *sync.RWMutex
	subs []*MySubscription
}

type MyPubSub struct {
	subjs  map[string]queue
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

func NewPubSub() PubSub {
	ctx, cancel := context.WithCancel(context.Background())
	return &MyPubSub{
		subjs:  make(map[string]queue),
		ctx:    ctx,
		cancel: cancel,
		wg:     new(sync.WaitGroup),
	}
}

func (p *MyPubSub) Subscribe(subj string, cb MsgHandler) (Subscription, error) {
	select {
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	default:
	}

	p.mu.Lock()
	if p.subjs == nil {
		p.mu.Unlock()
		return nil, context.Canceled
	}
	if _, ok := p.subjs[subj]; !ok {
		p.subjs[subj] = queue{mu: new(sync.RWMutex)}
	}
	p.mu.Unlock()

	in := make(chan any, 100)
	out := make(chan any, 100)
	ctx, cancel := context.WithCancel(p.ctx)
	p.wg.Go(func() { dynBuffer(ctx, in, out) })

	sub := MySubscription{subj: subj, cb: cb,
		ps: p, wrkCtx: ctx,
		wrkCancel: cancel, in: in, out: out}
	sub.isAlive.Store(true)

	p.wg.Go(func() {
		for {
			select {
			case val, ok := <-sub.out:
				if !ok {
					return
				}
				if sub.isAlive.Load() {
					sub.cb(val)
				}
			}
		}
	})

	p.mu.Lock()
	if p.subjs == nil {
		close(in)
		p.mu.Unlock()
		return nil, context.Canceled
	}
	added := p.subjs[subj]
	added.mu.Lock()
	added.subs = append(added.subs, &sub)
	p.subjs[subj] = added
	added.mu.Unlock()
	p.mu.Unlock()

	return &sub, nil
}

func (p *MyPubSub) Publish(subj string, msg interface{}) error {
	select {
	case <-p.ctx.Done():
		return p.ctx.Err()
	default:
	}

	p.mu.RLock()
	if p.subjs == nil {
		p.mu.RUnlock()
		return context.Canceled
	}
	q, ok := p.subjs[subj]
	if !ok {
		p.mu.RUnlock()
		return nil
	}
	q.mu.RLock()
	actQ := p.subjs[subj]
	p.mu.RUnlock()

	for _, sub := range actQ.subs {
		sub.in <- msg
	}
	q.mu.RUnlock()

	return nil
}

func (p *MyPubSub) Close(ctx context.Context) error {
	p.cancel()

	p.mu.Lock()
	for _, q := range p.subjs {
		q.mu.Lock()
		for _, sub := range q.subs {
			close(sub.in)
		}
		q.mu.Unlock()
	}
	p.subjs = nil
	p.mu.Unlock()

	waitWG := make(chan int)
	go func() {
		p.wg.Wait()
		close(waitWG)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-waitWG:
		return nil
	}
}

func write(ctx context.Context, buf []any, out chan<- any) error {
	for len(buf) > 0 {
		select {
		case out <- buf[0]:
			buf[0] = nil
			buf = buf[1:]
		}
	}
	close(out)
	return nil
}

func dynBuffer(ctx context.Context, in <-chan any, out chan<- any) error {
	var buf []any
	for {
		if len(buf) > 0 {
			select {
			case val, ok := <-in:
				if !ok {
					return write(ctx, buf, out)
				}
				buf = append(buf, val)
			case out <- buf[0]:
				buf[0] = nil
				buf = buf[1:]
			}
		} else {
			select {
			case val, ok := <-in:
				if !ok {
					close(out)
					return nil
				}
				buf = append(buf, val)
			}
		}
	}
}
