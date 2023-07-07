package libp2p

import (
	"github.com/kalyan3104/k-core/core"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	p2p "github.com/kalyan3104/k-core-p2p"
)

// ConnectionMonitor defines the behavior of a connection monitor
type ConnectionMonitor interface {
	network.Notifiee
	IsConnectedToTheNetwork(netw network.Network) bool
	SetThresholdMinConnectedPeers(thresholdMinConnectedPeers int, netw network.Network)
	ThresholdMinConnectedPeers() int
	Close() error
	IsInterfaceNil() bool
}

// PeerDiscovererWithSharder extends the PeerDiscoverer with the possibility to set the sharder
type PeerDiscovererWithSharder interface {
	p2p.PeerDiscoverer
	SetSharder(sharder p2p.Sharder) error
}

type p2pSigner interface {
	Sign(payload []byte) ([]byte, error)
	Verify(payload []byte, pid core.PeerID, signature []byte) error
	SignUsingPrivateKey(skBytes []byte, payload []byte) ([]byte, error)
}

// SendableData represents the struct used in data throttler implementation
type SendableData struct {
	Buff  []byte
	Topic string
	Sk    crypto.PrivKey
	ID    peer.ID
}

// ChannelLoadBalancer defines what a load balancer that uses chans should do
type ChannelLoadBalancer interface {
	AddChannel(channel string) error
	RemoveChannel(channel string) error
	GetChannelOrDefault(channel string) chan *SendableData
	CollectOneElementFromChannels() *SendableData
	Close() error
	IsInterfaceNil() bool
}
