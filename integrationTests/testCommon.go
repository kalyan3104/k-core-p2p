package integrationTests

import (
	"fmt"
	"strings"
	"time"

	logger "github.com/kalyan3104/k-core-logger-go"
	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/kalyan3104/k-core-p2p/config"
	"github.com/kalyan3104/k-core-p2p/libp2p"
	"github.com/kalyan3104/k-core-p2p/mock"
	"github.com/kalyan3104/k-core/marshal"
)

var log = logger.GetOrCreate("integrationtests")

// TestMarshaller GogoProtoMarshalizer used for tests
var TestMarshaller = &marshal.GogoProtoMarshalizer{}

// P2pBootstrapDelay is used so that nodes have enough time to bootstrap
var P2pBootstrapDelay = 5 * time.Second

func createP2PConfig(initialPeerList []string) config.P2PConfig {
	return config.P2PConfig{
		Node: config.NodeConfig{
			Port: "0",
		},
		KadDhtPeerDiscovery: config.KadDhtPeerDiscoveryConfig{
			Enabled:                          true,
			Type:                             "optimized",
			RefreshIntervalInSec:             2,
			ProtocolID:                       "/erd/kad/1.0.0",
			InitialPeerList:                  initialPeerList,
			BucketSize:                       100,
			RoutingTableRefreshIntervalInSec: 100,
		},
		Sharding: config.ShardingConfig{
			Type: p2p.NilListSharder,
		},
	}
}

// GetConnectableAddress returns a non circuit, non windows default connectable address for provided messenger
func GetConnectableAddress(mes p2p.Messenger) string {
	for _, addr := range mes.Addresses() {
		if strings.Contains(addr, "circuit") || strings.Contains(addr, "169.254") {
			continue
		}
		return addr
	}
	return ""
}

// WaitForBootstrapAndShowConnected will delay a given duration in order to wait for bootstraping  and print the
// number of peers that each node is connected to
func WaitForBootstrapAndShowConnected(peers []p2p.Messenger, durationBootstrapingTime time.Duration) {
	log.Info("Waiting for peer discovery...", "time", durationBootstrapingTime)
	time.Sleep(durationBootstrapingTime)

	strs := []string{"Connected peers:"}
	for _, peer := range peers {
		strs = append(strs, fmt.Sprintf("Peer %s is connected to %d peers", peer.ID().Pretty(), len(peer.ConnectedPeers())))
	}

	log.Info(strings.Join(strs, "\n"))
}

// CreateFixedNetworkOf8Peers assembles a network as following:
//
//	                     0------------------- 1
//	                     |                    |
//	2 ------------------ 3 ------------------ 4
//	|                    |                    |
//	5                    6                    7
func CreateFixedNetworkOf8Peers() ([]p2p.Messenger, error) {
	peers := createMessengersWithNoDiscovery(8)

	connections := map[int][]int{
		0: {1, 3},
		1: {4},
		2: {5, 3},
		3: {4, 6},
		4: {7},
	}

	err := createConnections(peers, connections)
	if err != nil {
		return nil, err
	}

	return peers, nil
}

func createMessengersWithNoDiscovery(numPeers int) []p2p.Messenger {
	peers := make([]p2p.Messenger, numPeers)

	for i := 0; i < numPeers; i++ {
		peers[i] = CreateMessengerWithNoDiscovery()
	}

	return peers
}

func createConnections(peers []p2p.Messenger, connections map[int][]int) error {
	for pid, connectTo := range connections {
		err := connectPeerToOthers(peers, pid, connectTo)
		if err != nil {
			return err
		}
	}

	return nil
}

func connectPeerToOthers(peers []p2p.Messenger, idx int, connectToIdxes []int) error {
	for _, connectToIdx := range connectToIdxes {
		err := peers[idx].ConnectToPeer(peers[connectToIdx].Addresses()[0])
		if err != nil {
			return fmt.Errorf("%w connecting %s to %s", err, peers[idx].ID(), peers[connectToIdx].ID())
		}
	}

	return nil
}

// CreateMessengerFromConfig creates a new libp2p messenger with provided configuration
func CreateMessengerFromConfig(p2pConfig config.P2PConfig) p2p.Messenger {
	arg := libp2p.ArgsNetworkMessenger{
		Marshalizer:           TestMarshaller,
		ListenAddress:         libp2p.TestListenAddrWithIp4AndTcp,
		P2pConfig:             p2pConfig,
		SyncTimer:             &libp2p.LocalSyncTimer{},
		PreferredPeersHolder:  &mock.PeersHolderStub{},
		NodeOperationMode:     p2p.NormalOperation,
		PeersRatingHandler:    &mock.PeersRatingHandlerStub{},
		ConnectionWatcherType: p2p.ConnectionWatcherTypePrint,
		P2pPrivateKey:         mock.NewPrivateKeyMock(),
		P2pSingleSigner:       &mock.SingleSignerStub{},
		P2pKeyGenerator:       &mock.KeyGenStub{},
	}

	if p2pConfig.Sharding.AdditionalConnections.MaxFullHistoryObservers > 0 {
		// we deliberately set this, automatically choose full archive node mode
		arg.NodeOperationMode = p2p.FullArchiveMode
	}

	libP2PMes, err := libp2p.NewNetworkMessenger(arg)
	log.LogIfError(err)

	return libP2PMes
}

// CreateMessengerWithNoDiscovery creates a new libp2p messenger with no peer discovery
func CreateMessengerWithNoDiscovery() p2p.Messenger {
	p2pCfg := createP2PConfigWithNoDiscovery()

	return CreateMessengerFromConfig(p2pCfg)
}

// createP2PConfigWithNoDiscovery creates a new libp2p messenger with no peer discovery
func createP2PConfigWithNoDiscovery() config.P2PConfig {
	return config.P2PConfig{
		Node: config.NodeConfig{
			Port: "0",
		},
		KadDhtPeerDiscovery: config.KadDhtPeerDiscoveryConfig{
			Enabled: false,
		},
		Sharding: config.ShardingConfig{
			Type: p2p.NilListSharder,
		},
	}
}

// CreateMessengerWithKadDht creates a new libp2p messenger with kad-dht peer discovery
func CreateMessengerWithKadDht(initialAddr string) p2p.Messenger {
	initialAddresses := make([]string, 0)
	if len(initialAddr) > 0 {
		initialAddresses = append(initialAddresses, initialAddr)
	}

	p2pCfg := createP2PConfig(initialAddresses)

	return CreateMessengerFromConfig(p2pCfg)
}

// CreateMessengerWithKadDhtAndProtocolID creates a new libp2p messenger with kad-dht peer discovery and peer ID
func CreateMessengerWithKadDhtAndProtocolID(initialAddr string, protocolID string) p2p.Messenger {
	initialAddresses := make([]string, 0)
	if len(initialAddr) > 0 {
		initialAddresses = append(initialAddresses, initialAddr)
	}
	p2pCfg := createP2PConfig(initialAddresses)
	p2pCfg.KadDhtPeerDiscovery.ProtocolID = protocolID

	return CreateMessengerFromConfig(p2pCfg)
}

// ClosePeers calls Messenger.Close on the provided peers
func ClosePeers(peers []p2p.Messenger) {
	for _, p := range peers {
		_ = p.Close()
	}
}

// IsIntInSlice returns true if idx is found on any position in the provided slice
func IsIntInSlice(idx int, slice []int) bool {
	for _, value := range slice {
		if value == idx {
			return true
		}
	}

	return false
}
