//go:build !solution

package gossip

import (
	"context"
	"maps"
	"sync"
	"time"

	"gitlab.com/slon/shad-go/gossip/meshpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PeerConfig struct {
	SelfEndpoint string
	PingPeriod   time.Duration
}

type Peer struct {
	meshpb.UnimplementedGossipServiceServer
	config   PeerConfig
	Snapshot map[string]*meshpb.PeerMeta
	musnap   *sync.RWMutex
	muconns  *sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       *sync.WaitGroup
	conns    map[string]*grpc.ClientConn
}

func (p *Peer) UpdateMeta(meta *meshpb.PeerMeta) {
	p.musnap.Lock()
	p.Snapshot[p.config.SelfEndpoint] = meta
	p.musnap.Unlock()

	p.muconns.RLock()
	for endpoint, conn := range p.conns {
		cli := meshpb.NewGossipServiceClient(conn)
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			ctx, cancel := context.WithTimeout(p.ctx, p.config.PingPeriod)
			defer cancel()
			_, err := cli.Update(ctx, &meshpb.NewPeerData{PeerMeta: meta,
				PeerEndpoint: p.config.SelfEndpoint})
			p.checkService(endpoint, err)
		}()
	}
	p.muconns.RUnlock()
}

func (p *Peer) getOrCreateConn(endpoint string) (*grpc.ClientConn, error) {
	p.muconns.RLock()
	conn, ok := p.conns[endpoint]
	p.muconns.RUnlock()
	if ok {
		return conn, nil
	}

	p.muconns.Lock()
	defer p.muconns.Unlock()

	if conn, ok := p.conns[endpoint]; ok {
		return conn, nil
	}

	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	p.conns[endpoint] = conn
	return conn, nil
}

func (p *Peer) AddSeed(seed string) {
	if seed == p.config.SelfEndpoint {
		return
	}

	conn, err := p.getOrCreateConn(seed)
	if err != nil {
		return
	}

	cli := meshpb.NewGossipServiceClient(conn)
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ctx, cancel := context.WithTimeout(p.ctx, p.config.PingPeriod)
		defer cancel()
		_, err = cli.Ping(ctx, &meshpb.EmptyData{})
		if err != nil {
			p.checkService(seed, err)
			return
		}
		p.musnap.Lock()
		if _, ok := p.Snapshot[seed]; !ok {
			p.Snapshot[seed] = &meshpb.PeerMeta{}
		}
		p.musnap.Unlock()
	}()
}

func (p *Peer) Addr() string {
	return p.config.SelfEndpoint
}

func (p *Peer) GetMembers() map[string]*meshpb.PeerMeta {
	p.musnap.RLock()
	defer p.musnap.RUnlock()
	return maps.Clone(p.Snapshot)
}

func (p *Peer) checkService(endpoint string, err error) {
	if err == nil {
		return
	}

	p.muconns.Lock()
	conn, ok := p.conns[endpoint]
	if ok && conn != nil {
		conn.Close()
		delete(p.conns, endpoint)
	}
	p.muconns.Unlock()

	p.musnap.Lock()
	delete(p.Snapshot, endpoint)
	p.musnap.Unlock()
}

func (p *Peer) ping(endpoint string, conn *grpc.ClientConn) {
	cli := meshpb.NewGossipServiceClient(conn)
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ctx, cancel := context.WithTimeout(p.ctx, p.config.PingPeriod)
		defer cancel()
		_, err := cli.Ping(ctx, &meshpb.EmptyData{})
		if err != nil {
			p.checkService(endpoint, err)
			return
		}
		p.musnap.RLock()
		snap := maps.Clone(p.Snapshot)
		p.musnap.RUnlock()

		for ep, meta := range snap {
			ctx2, cancel2 := context.WithTimeout(p.ctx, p.config.PingPeriod)
			_, _ = cli.Update(ctx2, &meshpb.NewPeerData{PeerEndpoint: ep, PeerMeta: meta})
			cancel2()
		}
	}()
}

func (p *Peer) pingAll() {
	p.muconns.RLock()
	for endpoint, conn := range p.conns {
		p.ping(endpoint, conn)
	}
	p.muconns.RUnlock()

}

func (p *Peer) Run() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(p.config.PingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				p.pingAll()
			case <-p.ctx.Done():
				return
			}
		}
	}()

}

func (p *Peer) Stop() {
	p.cancel()
	p.wg.Wait()

	for _, conn := range p.conns {
		conn.Close()
	}

}

func NewPeer(config PeerConfig) *Peer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Peer{
		config:   config,
		Snapshot: map[string]*meshpb.PeerMeta{config.SelfEndpoint: {}},
		musnap:   &sync.RWMutex{},
		muconns:  &sync.RWMutex{},
		ctx:      ctx,
		cancel:   cancel,
		wg:       &sync.WaitGroup{},
		conns:    make(map[string]*grpc.ClientConn),
	}
}

func (p *Peer) Ping(ctx context.Context, data *meshpb.EmptyData) (*meshpb.EmptyData, error) {
	return &meshpb.EmptyData{}, nil
}

func (p *Peer) Update(ctx context.Context, data *meshpb.NewPeerData) (*meshpb.EmptyData, error) {
	if data.GetPeerEndpoint() == p.config.SelfEndpoint {
		return &meshpb.EmptyData{}, nil
	}

	p.musnap.Lock()
	val, ok := p.Snapshot[data.GetPeerEndpoint()]
	if ok && val.Name == data.GetPeerMeta().Name {
		p.musnap.Unlock()
		return &meshpb.EmptyData{}, nil
	}
	if ok {
		p.Snapshot[data.GetPeerEndpoint()] = data.GetPeerMeta()
	}
	p.musnap.Unlock()

	if ok {
		p.muconns.RLock()
		for endpoint, conn := range p.conns {
			if endpoint == data.GetPeerEndpoint() {
				continue
			}

			p.wg.Add(1)
			go func() {
				defer p.wg.Done()
				ctx, cancel := context.WithTimeout(p.ctx, p.config.PingPeriod)
				defer cancel()
				cli := meshpb.NewGossipServiceClient(conn)
				_, err := cli.Update(ctx, data)
				if err != nil {
					p.checkService(endpoint, err)
				}
			}()
		}
		p.muconns.RUnlock()
		return &meshpb.EmptyData{}, nil
	}

	conn, err := p.getOrCreateConn(data.GetPeerEndpoint())
	if err != nil {
		return nil, err
	}

	cli := meshpb.NewGossipServiceClient(conn)
	ctxPing, cancel := context.WithTimeout(p.ctx, p.config.PingPeriod)
	defer cancel()
	_, err = cli.Ping(ctxPing, &meshpb.EmptyData{})
	if err != nil {
		p.checkService(data.GetPeerEndpoint(), err)
		return &meshpb.EmptyData{}, nil
	}
	p.musnap.Lock()
	if _, ok := p.Snapshot[data.GetPeerEndpoint()]; !ok {
		p.Snapshot[data.GetPeerEndpoint()] = data.GetPeerMeta()
	}
	p.musnap.Unlock()

	p.muconns.RLock()
	for endpoint, conn := range p.conns {
		if endpoint == data.GetPeerEndpoint() {
			continue
		}

		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			ctx, cancel := context.WithTimeout(p.ctx, p.config.PingPeriod)
			defer cancel()
			cli := meshpb.NewGossipServiceClient(conn)
			_, err := cli.Update(ctx, data)
			if err != nil {
				p.checkService(endpoint, err)
			}
		}()
	}
	p.muconns.RUnlock()

	return &meshpb.EmptyData{}, nil
}
