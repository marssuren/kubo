package libp2p

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p"
	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/routing"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	multiaddr "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core/node/helpers"
	"github.com/ipfs/kubo/repo"

	"go.uber.org/fx"
)

type P2PHostIn struct {
	fx.In

	Repo          repo.Repo
	Validator     record.Validator
	HostOption    HostOption
	RoutingOption RoutingOption
	ID            peer.ID
	Peerstore     peerstore.Peerstore

	Opts [][]libp2p.Option `group:"libp2p"`
}

type P2PHostOut struct {
	fx.Out

	Host    host.Host
	Routing routing.Routing `name:"initialrouting"`
}

type simpleConnGater struct {
	lock     sync.Mutex
	blockAll bool
}

func (s *simpleConnGater) SetBlockAll(blockAll bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.blockAll = blockAll
}

func (s *simpleConnGater) InterceptAccept(network.ConnMultiaddrs) (allow bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return !s.blockAll
}

// InterceptAddrDial implements connmgr.ConnectionGater.
func (s *simpleConnGater) InterceptAddrDial(peer.ID, multiaddr.Multiaddr) (allow bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return !s.blockAll
}

// InterceptPeerDial implements connmgr.ConnectionGater.
func (s *simpleConnGater) InterceptPeerDial(p peer.ID) (allow bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return !s.blockAll
}

// InterceptSecured implements connmgr.ConnectionGater.
func (s *simpleConnGater) InterceptSecured(network.Direction, peer.ID, network.ConnMultiaddrs) (allow bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return !s.blockAll
}

// InterceptUpgraded implements connmgr.ConnectionGater.
func (s *simpleConnGater) InterceptUpgraded(network.Conn) (allow bool, reason control.DisconnectReason) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return !s.blockAll, 0
}

var _ connmgr.ConnectionGater = (*simpleConnGater)(nil)

var DebugConnGater = &simpleConnGater{}

func Host(mctx helpers.MetricsCtx, lc fx.Lifecycle, params P2PHostIn) (out P2PHostOut, err error) {
	opts := []libp2p.Option{libp2p.NoListenAddrs, libp2p.ConnectionGater(DebugConnGater)}
	for _, o := range params.Opts {
		opts = append(opts, o...)
	}

	ctx := helpers.LifecycleCtx(mctx, lc)
	cfg, err := params.Repo.Config()
	if err != nil {
		return out, err
	}
	bootstrappers, err := cfg.BootstrapPeers()
	if err != nil {
		return out, err
	}

	routingOptArgs := RoutingOptionArgs{
		Ctx:                           ctx,
		Datastore:                     params.Repo.Datastore(),
		Validator:                     params.Validator,
		BootstrapPeers:                bootstrappers,
		OptimisticProvide:             cfg.Experimental.OptimisticProvide,
		OptimisticProvideJobsPoolSize: cfg.Experimental.OptimisticProvideJobsPoolSize,
		LoopbackAddressesOnLanDHT:     cfg.Routing.LoopbackAddressesOnLanDHT.WithDefault(config.DefaultLoopbackAddressesOnLanDHT),
	}
	opts = append(opts, libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
		args := routingOptArgs
		args.Host = h
		r, err := params.RoutingOption(args)
		out.Routing = r
		return r, err
	}))

	out.Host, err = params.HostOption(params.ID, params.Peerstore, opts...)
	if err != nil {
		return P2PHostOut{}, err
	}

	routingOptArgs.Host = out.Host

	// this code is necessary just for tests: mock network constructions
	// ignore the libp2p constructor options that actually construct the routing!
	if out.Routing == nil {
		r, err := params.RoutingOption(routingOptArgs)
		if err != nil {
			return P2PHostOut{}, err
		}
		out.Routing = r
		out.Host = routedhost.Wrap(out.Host, out.Routing)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return out.Host.Close()
		},
	})

	return out, err
}
