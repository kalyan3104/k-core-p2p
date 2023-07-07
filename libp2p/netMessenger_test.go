package libp2p_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	logger "github.com/kalyan3104/k-core-logger-go"
	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/kalyan3104/k-core-p2p/config"
	"github.com/kalyan3104/k-core-p2p/data"
	"github.com/kalyan3104/k-core-p2p/libp2p"
	p2pCrypto "github.com/kalyan3104/k-core-p2p/libp2p/crypto"
	"github.com/kalyan3104/k-core-p2p/message"
	"github.com/kalyan3104/k-core-p2p/mock"
	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/kalyan3104/k-core/marshal"
	commonCrypto "github.com/kalyan3104/k-crypto-core-go"
	"github.com/kalyan3104/k-crypto-core-go/signing"
	"github.com/kalyan3104/k-crypto-core-go/signing/secp256k1"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTopic = "test"

var timeoutWaitResponses = time.Second * 2
var log = logger.GetOrCreate("p2p/libp2p/tests")
var noSigCheckHandler = func(sigSize int) bool { return true }

func closeMessengers(messengers ...p2p.Messenger) {
	for _, mes := range messengers {
		_ = mes.Close()
	}
}

type noSigner struct {
	p2p.SignerVerifier
}

// Sign -
func (signer *noSigner) Sign(_ []byte) ([]byte, error) {
	return make([]byte, 0), nil
}

func waitDoneWithTimeout(t *testing.T, chanDone chan bool, timeout time.Duration) {
	select {
	case <-chanDone:
		return
	case <-time.After(timeout):
		assert.Fail(t, "timeout reached")
	}
}

func prepareMessengerForMatchDataReceive(messenger p2p.Messenger, matchData []byte, wg *sync.WaitGroup, checkSigSize func(sigSize int) bool) {
	err := messenger.CreateTopic(testTopic, false)

	err = messenger.RegisterMessageProcessor(testTopic, "identifier",
		&mock.MessageProcessorStub{
			ProcessMessageCalled: func(message p2p.MessageP2P, _ core.PeerID) error {
				if !bytes.Equal(matchData, message.Data()) {
					return nil
				}
				if !checkSigSize(len(message.Signature())) {
					return nil
				}

				// do not print the message.Data() or matchData as the test TestLibp2pMessenger_BroadcastDataBetween2PeersWithLargeMsgShouldWork
				// will cause disruption on the github action page

				wg.Done()

				return nil
			},
		})
	_ = err
}

func getConnectableAddress(messenger p2p.Messenger) string {
	for _, addr := range messenger.Addresses() {
		if strings.Contains(addr, "circuit") || strings.Contains(addr, "169.254") {
			continue
		}

		return addr
	}

	return ""
}

func createMockNetworkArgs() libp2p.ArgsNetworkMessenger {
	return libp2p.ArgsNetworkMessenger{
		Marshalizer:   &mock.ProtoMarshallerMock{},
		ListenAddress: libp2p.TestListenAddrWithIp4AndTcp,
		P2pConfig: config.P2PConfig{
			Node: config.NodeConfig{
				Port: "0",
			},
			KadDhtPeerDiscovery: config.KadDhtPeerDiscoveryConfig{
				Enabled: false,
			},
			Sharding: config.ShardingConfig{
				Type: p2p.NilListSharder,
			},
		},
		SyncTimer:             &libp2p.LocalSyncTimer{},
		PreferredPeersHolder:  &mock.PeersHolderStub{},
		PeersRatingHandler:    &mock.PeersRatingHandlerStub{},
		ConnectionWatcherType: p2p.ConnectionWatcherTypePrint,
		P2pPrivateKey:         mock.NewPrivateKeyMock(),
		P2pSingleSigner:       &mock.SingleSignerStub{},
		P2pKeyGenerator:       &mock.KeyGenStub{},
	}
}

func createMockNetworkOf2() (mocknet.Mocknet, p2p.Messenger, p2p.Messenger) {
	netw := mocknet.New()

	messenger1, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	messenger2, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)

	_ = netw.LinkAll()

	return netw, messenger1, messenger2
}

func createMockNetworkOf3() (p2p.Messenger, p2p.Messenger, p2p.Messenger) {
	netw := mocknet.New()

	messenger1, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	messenger2, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	messenger3, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)

	_ = netw.LinkAll()

	nscm1 := mock.NewNetworkShardingCollectorMock()
	nscm1.PutPeerIdSubType(messenger1.ID(), core.FullHistoryObserver)
	nscm1.PutPeerIdSubType(messenger2.ID(), core.FullHistoryObserver)
	nscm1.PutPeerIdSubType(messenger3.ID(), core.RegularPeer)
	_ = messenger1.SetPeerShardResolver(nscm1)

	nscm2 := mock.NewNetworkShardingCollectorMock()
	nscm2.PutPeerIdSubType(messenger1.ID(), core.FullHistoryObserver)
	nscm2.PutPeerIdSubType(messenger2.ID(), core.FullHistoryObserver)
	nscm2.PutPeerIdSubType(messenger3.ID(), core.RegularPeer)
	_ = messenger2.SetPeerShardResolver(nscm2)

	nscm3 := mock.NewNetworkShardingCollectorMock()
	nscm3.PutPeerIdSubType(messenger1.ID(), core.FullHistoryObserver)
	nscm3.PutPeerIdSubType(messenger2.ID(), core.FullHistoryObserver)
	nscm3.PutPeerIdSubType(messenger3.ID(), core.RegularPeer)
	_ = messenger3.SetPeerShardResolver(nscm3)

	return messenger1, messenger2, messenger3
}

func createMockMessenger() p2p.Messenger {
	netw := mocknet.New()

	messenger, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)

	return messenger
}

func containsPeerID(list []core.PeerID, searchFor core.PeerID) bool {
	for _, pid := range list {
		if bytes.Equal(pid.Bytes(), searchFor.Bytes()) {
			return true
		}
	}
	return false
}

// ------- NewMemoryLibp2pMessenger

func TestNewMemoryLibp2pMessenger_NilMockNetShouldErr(t *testing.T) {
	args := createMockNetworkArgs()
	messenger, err := libp2p.NewMockMessenger(args, nil)

	assert.Nil(t, messenger)
	assert.Equal(t, p2p.ErrNilMockNet, err)
}

func TestNewMemoryLibp2pMessenger_OkValsWithoutDiscoveryShouldWork(t *testing.T) {
	netw := mocknet.New()

	messenger, err := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	defer closeMessengers(messenger)

	assert.Nil(t, err)
	assert.False(t, check.IfNil(messenger))
}

// ------- NewNetworkMessenger

func TestNewNetworkMessenger_NilChecksShouldErr(t *testing.T) {
	t.Parallel()

	t.Run("nil messenger", func(t *testing.T) {
		t.Parallel()

		arg := createMockNetworkArgs()
		arg.Marshalizer = nil
		messenger, err := libp2p.NewNetworkMessenger(arg)

		assert.True(t, check.IfNil(messenger))
		assert.True(t, errors.Is(err, p2p.ErrNilMarshalizer))
	})

	t.Run("nil preferred peers holder", func(t *testing.T) {
		t.Parallel()

		arg := createMockNetworkArgs()
		arg.PreferredPeersHolder = nil
		messenger, err := libp2p.NewNetworkMessenger(arg)

		assert.True(t, check.IfNil(messenger))
		assert.True(t, errors.Is(err, p2p.ErrNilPreferredPeersHolder))
	})

	t.Run("nil peers rating handler", func(t *testing.T) {
		t.Parallel()

		arg := createMockNetworkArgs()
		arg.PeersRatingHandler = nil
		mes, err := libp2p.NewNetworkMessenger(arg)

		assert.True(t, check.IfNil(mes))
		assert.True(t, errors.Is(err, p2p.ErrNilPeersRatingHandler))
	})

	t.Run("nil sync timer", func(t *testing.T) {
		t.Parallel()

		arg := createMockNetworkArgs()
		arg.SyncTimer = nil
		messenger, err := libp2p.NewNetworkMessenger(arg)

		assert.True(t, check.IfNil(messenger))
		assert.True(t, errors.Is(err, p2p.ErrNilSyncTimer))
	})

	t.Run("nil p2p private key", func(t *testing.T) {
		t.Parallel()

		arg := createMockNetworkArgs()
		arg.P2pPrivateKey = nil
		messenger, err := libp2p.NewNetworkMessenger(arg)

		assert.True(t, check.IfNil(messenger))
		assert.True(t, errors.Is(err, p2p.ErrNilP2pPrivateKey))
	})

	t.Run("nil p2p single signer", func(t *testing.T) {
		t.Parallel()

		arg := createMockNetworkArgs()
		arg.P2pSingleSigner = nil
		messenger, err := libp2p.NewNetworkMessenger(arg)

		assert.True(t, check.IfNil(messenger))
		assert.True(t, errors.Is(err, p2p.ErrNilP2pSingleSigner))
	})

	t.Run("nil p2p key generator", func(t *testing.T) {
		t.Parallel()

		arg := createMockNetworkArgs()
		arg.P2pKeyGenerator = nil
		messenger, err := libp2p.NewNetworkMessenger(arg)

		assert.True(t, check.IfNil(messenger))
		assert.True(t, errors.Is(err, p2p.ErrNilP2pKeyGenerator))
	})
}

func TestNewNetworkMessenger_WithDeactivatedKadDiscovererShouldWork(t *testing.T) {
	arg := createMockNetworkArgs()
	messenger, err := libp2p.NewNetworkMessenger(arg)
	defer closeMessengers(messenger)

	assert.NotNil(t, messenger)
	assert.Nil(t, err)
}

func TestNewNetworkMessenger_WithKadDiscovererListsSharderInvalidTargetConnShouldErr(t *testing.T) {
	arg := createMockNetworkArgs()
	arg.P2pConfig.KadDhtPeerDiscovery = config.KadDhtPeerDiscoveryConfig{
		Enabled:                          true,
		Type:                             "optimized",
		RefreshIntervalInSec:             10,
		ProtocolID:                       "/erd/kad/1.0.0",
		InitialPeerList:                  nil,
		BucketSize:                       100,
		RoutingTableRefreshIntervalInSec: 10,
	}
	arg.P2pConfig.Sharding.Type = p2p.ListsSharder
	messenger, err := libp2p.NewNetworkMessenger(arg)

	assert.True(t, check.IfNil(messenger))
	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
}

func TestNewNetworkMessenger_WithKadDiscovererListSharderShouldWork(t *testing.T) {
	arg := createMockNetworkArgs()
	arg.P2pConfig.KadDhtPeerDiscovery = config.KadDhtPeerDiscoveryConfig{
		Enabled:                          true,
		Type:                             "optimized",
		RefreshIntervalInSec:             10,
		ProtocolID:                       "/erd/kad/1.0.0",
		InitialPeerList:                  nil,
		BucketSize:                       100,
		RoutingTableRefreshIntervalInSec: 10,
	}
	arg.P2pConfig.Sharding = config.ShardingConfig{
		Type:            p2p.NilListSharder,
		TargetPeerCount: 10,
	}
	messenger, err := libp2p.NewNetworkMessenger(arg)
	defer closeMessengers(messenger)

	assert.False(t, check.IfNil(messenger))
	assert.Nil(t, err)
}

func TestNewNetworkMessenger_WithListenAddrWithIp4AndTcpShouldWork(t *testing.T) {
	arg := createMockNetworkArgs()
	arg.ListenAddress = libp2p.TestListenAddrWithIp4AndTcp
	arg.P2pConfig.KadDhtPeerDiscovery = config.KadDhtPeerDiscoveryConfig{
		Enabled:                          true,
		Type:                             "optimized",
		RefreshIntervalInSec:             10,
		ProtocolID:                       "/erd/kad/1.0.0",
		InitialPeerList:                  nil,
		BucketSize:                       100,
		RoutingTableRefreshIntervalInSec: 10,
	}
	arg.P2pConfig.Sharding = config.ShardingConfig{
		Type:            p2p.NilListSharder,
		TargetPeerCount: 10,
	}
	messenger, err := libp2p.NewNetworkMessenger(arg)
	defer closeMessengers(messenger)

	assert.False(t, check.IfNil(messenger))
	assert.Nil(t, err)
}

// ------- Messenger functionality

func TestLibp2pMessenger_ConnectToPeerShouldCallUpgradedHost(t *testing.T) {
	netw := mocknet.New()

	messenger, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	closeMessengers(messenger)

	wasCalled := false

	p := "peer"

	uhs := &mock.ConnectableHostStub{
		ConnectToPeerCalled: func(ctx context.Context, address string) error {
			if p == address {
				wasCalled = true
			}
			return nil
		},
	}

	messenger.SetHost(uhs)
	_ = messenger.ConnectToPeer(p)
	assert.True(t, wasCalled)
}

func TestLibp2pMessenger_IsConnectedShouldWork(t *testing.T) {
	_, messenger1, messenger2 := createMockNetworkOf2()
	defer closeMessengers(messenger1, messenger2)

	adr2 := messenger2.Addresses()[0]

	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)

	assert.True(t, messenger1.IsConnected(messenger2.ID()))
	assert.True(t, messenger2.IsConnected(messenger1.ID()))
}

func TestLibp2pMessenger_CreateTopicOkValsShouldWork(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	err := messenger.CreateTopic("test", true)
	assert.Nil(t, err)
}

func TestLibp2pMessenger_CreateTopicTwiceShouldNotErr(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	_ = messenger.CreateTopic("test", false)
	err := messenger.CreateTopic("test", false)
	assert.Nil(t, err)
}

func TestLibp2pMessenger_HasTopicIfHaveTopicShouldReturnTrue(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	_ = messenger.CreateTopic("test", false)

	assert.True(t, messenger.HasTopic("test"))
}

func TestLibp2pMessenger_HasTopicIfDoNotHaveTopicShouldReturnFalse(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	_ = messenger.CreateTopic("test", false)

	assert.False(t, messenger.HasTopic("one topic"))
}

func TestLibp2pMessenger_RegisterTopicValidatorOnInexistentTopicShouldWork(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	err := messenger.RegisterMessageProcessor("test", "identifier", &mock.MessageProcessorStub{})

	assert.Nil(t, err)
}

func TestLibp2pMessenger_RegisterTopicValidatorWithNilHandlerShouldErr(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	_ = messenger.CreateTopic("test", false)

	err := messenger.RegisterMessageProcessor("test", "identifier", nil)

	assert.True(t, errors.Is(err, p2p.ErrNilValidator))
}

func TestLibp2pMessenger_RegisterTopicValidatorOkValsShouldWork(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	_ = messenger.CreateTopic("test", false)

	err := messenger.RegisterMessageProcessor("test", "identifier", &mock.MessageProcessorStub{})

	assert.Nil(t, err)
}

func TestLibp2pMessenger_RegisterTopicValidatorReregistrationShouldErr(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	_ = messenger.CreateTopic("test", false)
	// registration
	_ = messenger.RegisterMessageProcessor("test", "identifier", &mock.MessageProcessorStub{})
	// re-registration
	err := messenger.RegisterMessageProcessor("test", "identifier", &mock.MessageProcessorStub{})

	assert.True(t, errors.Is(err, p2p.ErrMessageProcessorAlreadyDefined))
}

func TestLibp2pMessenger_UnegisterTopicValidatorOnANotRegisteredTopicShouldNotErr(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	_ = messenger.CreateTopic("test", false)
	err := messenger.UnregisterMessageProcessor("test", "identifier")

	assert.Nil(t, err)
}

func TestLibp2pMessenger_UnregisterTopicValidatorShouldWork(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	_ = messenger.CreateTopic("test", false)

	// registration
	_ = messenger.RegisterMessageProcessor("test", "identifier", &mock.MessageProcessorStub{})

	// unregistration
	err := messenger.UnregisterMessageProcessor("test", "identifier")

	assert.Nil(t, err)
}

func TestLibp2pMessenger_UnregisterAllTopicValidatorShouldWork(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	_ = messenger.CreateTopic("test", false)
	// registration
	_ = messenger.CreateTopic("test1", false)
	_ = messenger.RegisterMessageProcessor("test1", "identifier", &mock.MessageProcessorStub{})
	_ = messenger.CreateTopic("test2", false)
	_ = messenger.RegisterMessageProcessor("test2", "identifier", &mock.MessageProcessorStub{})
	// unregistration
	err := messenger.UnregisterAllMessageProcessors()
	assert.Nil(t, err)
	err = messenger.RegisterMessageProcessor("test1", "identifier", &mock.MessageProcessorStub{})
	assert.Nil(t, err)
	err = messenger.RegisterMessageProcessor("test2", "identifier", &mock.MessageProcessorStub{})
	assert.Nil(t, err)
}

func TestLibp2pMessenger_RegisterUnregisterConcurrentlyShouldNotPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r != nil {
			assert.Fail(t, fmt.Sprintf("should have not panic: %v", r))
		}
	}()

	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	topic := "test topic"
	_ = messenger.CreateTopic(topic, false)

	numIdentifiers := 100
	identifiers := make([]string, 0, numIdentifiers)
	for i := 0; i < numIdentifiers; i++ {
		identifiers = append(identifiers, fmt.Sprintf("identifier%d", i))
	}

	wg := sync.WaitGroup{}
	wg.Add(numIdentifiers * 3)
	for i := 0; i < numIdentifiers; i++ {
		go func(index int) {
			_ = messenger.RegisterMessageProcessor(topic, identifiers[index], &mock.MessageProcessorStub{})
			wg.Done()
		}(i)

		go func(index int) {
			_ = messenger.UnregisterMessageProcessor(topic, identifiers[index])
			wg.Done()
		}(i)

		go func() {
			messenger.Broadcast(topic, []byte("buff"))
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestLibp2pMessenger_BroadcastDataLargeMessageShouldNotCallSend(t *testing.T) {
	msg := make([]byte, libp2p.MaxSendBuffSize+1)
	messenger, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())
	defer closeMessengers(messenger)

	messenger.SetLoadBalancer(&mock.ChannelLoadBalancerStub{
		GetChannelOrDefaultCalled: func(pipe string) chan *libp2p.SendableData {
			assert.Fail(t, "should have not got to this line")

			return make(chan *libp2p.SendableData, 1)
		},
		CollectOneElementFromChannelsCalled: func() *libp2p.SendableData {
			return nil
		},
	})

	messenger.Broadcast("topic", msg)
}

func TestLibp2pMessenger_BroadcastDataBetween2PeersShouldWork(t *testing.T) {
	msg := []byte("test message")

	_, messenger1, messenger2 := createMockNetworkOf2()
	defer closeMessengers(messenger1, messenger2)

	adr2 := messenger2.Addresses()[0]

	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)

	wg := &sync.WaitGroup{}
	chanDone := make(chan bool)
	wg.Add(2)

	go func() {
		wg.Wait()
		chanDone <- true
	}()

	prepareMessengerForMatchDataReceive(messenger1, msg, wg, noSigCheckHandler)
	prepareMessengerForMatchDataReceive(messenger2, msg, wg, noSigCheckHandler)

	fmt.Println("Delaying as to allow peers to announce themselves on the opened topic...")
	time.Sleep(time.Second)

	fmt.Printf("sending message from %s...\n", messenger1.ID().Pretty())

	messenger1.Broadcast("test", msg)

	waitDoneWithTimeout(t, chanDone, timeoutWaitResponses)
}

func TestLibp2pMessenger_BroadcastOnChannelBlockingShouldLimitNumberOfGoRoutines(t *testing.T) {
	if testing.Short() {
		t.Skip("this test does not perform well in TC with race detector on")
	}

	msg := []byte("test message")
	numBroadcasts := libp2p.BroadcastGoRoutines + 5

	ch := make(chan *libp2p.SendableData)

	wg := sync.WaitGroup{}
	wg.Add(numBroadcasts)

	messenger, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())
	defer closeMessengers(messenger)

	messenger.SetLoadBalancer(&mock.ChannelLoadBalancerStub{
		CollectOneElementFromChannelsCalled: func() *libp2p.SendableData {
			return nil
		},
		GetChannelOrDefaultCalled: func(pipe string) chan *libp2p.SendableData {
			wg.Done()
			return ch
		},
	})

	numErrors := uint32(0)

	for i := 0; i < numBroadcasts; i++ {
		go func() {
			err := messenger.BroadcastOnChannelBlocking("test", "test", msg)
			if err == p2p.ErrTooManyGoroutines {
				atomic.AddUint32(&numErrors, 1)
				wg.Done()
			}
		}()
	}

	wg.Wait()

	// cleanup stuck go routines that are trying to write on the ch channel
	for i := 0; i < libp2p.BroadcastGoRoutines; i++ {
		select {
		case <-ch:
		default:
		}
	}

	assert.True(t, atomic.LoadUint32(&numErrors) > 0)
}

func TestLibp2pMessenger_BroadcastDataBetween2PeersWithLargeMsgShouldWork(t *testing.T) {
	msg := bytes.Repeat([]byte{'A'}, libp2p.MaxSendBuffSize)

	_, messenger1, messenger2 := createMockNetworkOf2()
	defer closeMessengers(messenger1, messenger2)

	adr2 := messenger2.Addresses()[0]

	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)

	wg := &sync.WaitGroup{}
	chanDone := make(chan bool)
	wg.Add(2)

	go func() {
		wg.Wait()
		chanDone <- true
	}()

	prepareMessengerForMatchDataReceive(messenger1, msg, wg, noSigCheckHandler)
	prepareMessengerForMatchDataReceive(messenger2, msg, wg, noSigCheckHandler)

	fmt.Println("Delaying as to allow peers to announce themselves on the opened topic...")
	time.Sleep(time.Second)

	fmt.Printf("sending message from %s...\n", messenger1.ID().Pretty())

	messenger1.Broadcast("test", msg)

	waitDoneWithTimeout(t, chanDone, timeoutWaitResponses)
}

func TestLibp2pMessenger_Peers(t *testing.T) {
	_, messenger1, messenger2 := createMockNetworkOf2()
	defer closeMessengers(messenger1, messenger2)

	adr2 := messenger2.Addresses()[0]

	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)

	// should know both peers
	foundCurrent := false
	foundConnected := false

	for _, p := range messenger1.Peers() {
		fmt.Println(p.Pretty())

		if p.Pretty() == messenger1.ID().Pretty() {
			foundCurrent = true
		}
		if p.Pretty() == messenger2.ID().Pretty() {
			foundConnected = true
		}
	}

	assert.True(t, foundCurrent && foundConnected)
}

func TestLibp2pMessenger_ConnectedPeers(t *testing.T) {
	netw, messenger1, messenger2 := createMockNetworkOf2()
	messenger3, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	defer closeMessengers(messenger1, messenger2, messenger3)

	_ = netw.LinkAll()

	adr2 := messenger2.Addresses()[0]

	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)
	_ = messenger3.ConnectToPeer(adr2)

	// connected peers:  1 ----- 2 ----- 3

	assert.Equal(t, []core.PeerID{messenger2.ID()}, messenger1.ConnectedPeers())
	assert.Equal(t, []core.PeerID{messenger2.ID()}, messenger3.ConnectedPeers())
	assert.Equal(t, 2, len(messenger2.ConnectedPeers()))
	// no need to further test that messenger2 is connected to messenger1 and messenger3 as this was tested in first 2 asserts
}

func TestLibp2pMessenger_ConnectedAddresses(t *testing.T) {
	netw, messenger1, messenger2 := createMockNetworkOf2()
	messenger3, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	defer closeMessengers(messenger1, messenger2, messenger3)

	_ = netw.LinkAll()

	adr2 := messenger2.Addresses()[0]

	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)
	_ = messenger3.ConnectToPeer(adr2)

	// connected peers:  1 ----- 2 ----- 3

	foundAddr1 := false
	foundAddr3 := false

	for _, addr := range messenger2.ConnectedAddresses() {
		for _, address := range messenger1.Addresses() {
			if addr == address {
				foundAddr1 = true
			}
		}

		for _, address := range messenger3.Addresses() {
			if addr == address {
				foundAddr3 = true
			}
		}
	}

	assert.True(t, foundAddr1)
	assert.True(t, foundAddr3)
	assert.Equal(t, 2, len(messenger2.ConnectedAddresses()))
	// no need to further test that messenger2 is connected to messenger1 and messenger3 as this was tested in first 2 asserts
}

func TestLibp2pMessenger_PeerAddressConnectedPeerShouldWork(t *testing.T) {
	netw, messenger1, messenger2 := createMockNetworkOf2()
	messenger3, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	defer closeMessengers(messenger1, messenger2, messenger3)

	_ = netw.LinkAll()

	adr2 := messenger2.Addresses()[0]

	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)
	_ = messenger3.ConnectToPeer(adr2)

	// connected peers:  1 ----- 2 ----- 3

	addressesRecov := messenger2.PeerAddresses(messenger1.ID())
	for _, addr := range messenger1.Addresses() {
		for _, addrRecov := range addressesRecov {
			if strings.Contains(addr, addrRecov) {
				// address returned is valid, test is successful
				return
			}
		}
	}

	assert.Fail(t, "Returned address is not valid!")
}

func TestLibp2pMessenger_PeerAddressNotConnectedShouldReturnFromPeerstore(t *testing.T) {
	netw := mocknet.New()
	messenger, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	defer closeMessengers(messenger)

	networkHandler := &mock.NetworkStub{
		ConnsCalled: func() []network.Conn {
			return nil
		},
	}

	peerstoreHandler := &mock.PeerstoreStub{
		AddrsCalled: func(p peer.ID) []multiaddr.Multiaddr {
			return []multiaddr.Multiaddr{
				&mock.MultiaddrStub{
					StringCalled: func() string {
						return "multiaddress 1"
					},
				},
				&mock.MultiaddrStub{
					StringCalled: func() string {
						return "multiaddress 2"
					},
				},
			}
		},
	}

	messenger.SetHost(&mock.ConnectableHostStub{
		NetworkCalled: func() network.Network {
			return networkHandler
		},
		PeerstoreCalled: func() peerstore.Peerstore {
			return peerstoreHandler
		},
	})

	addresses := messenger.PeerAddresses("pid")
	require.Equal(t, 2, len(addresses))
	assert.Equal(t, addresses[0], "multiaddress 1")
	assert.Equal(t, addresses[1], "multiaddress 2")
}

func TestLibp2pMessenger_PeerAddressDisconnectedPeerShouldWork(t *testing.T) {
	netw, messenger1, messenger2 := createMockNetworkOf2()
	messenger3, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	defer closeMessengers(messenger1, messenger2, messenger3)

	_ = netw.LinkAll()

	adr2 := messenger2.Addresses()[0]

	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)
	_ = messenger3.ConnectToPeer(adr2)

	_ = netw.UnlinkPeers(peer.ID(messenger1.ID().Bytes()), peer.ID(messenger2.ID().Bytes()))
	_ = netw.DisconnectPeers(peer.ID(messenger1.ID().Bytes()), peer.ID(messenger2.ID().Bytes()))
	_ = netw.DisconnectPeers(peer.ID(messenger2.ID().Bytes()), peer.ID(messenger1.ID().Bytes()))

	// connected peers:  1 --x-- 2 ----- 3

	assert.False(t, messenger2.IsConnected(messenger1.ID()))
}

func TestLibp2pMessenger_PeerAddressUnknownPeerShouldReturnEmpty(t *testing.T) {
	_, messenger1, messenger2 := createMockNetworkOf2()
	defer closeMessengers(messenger1, messenger2)

	adr1Recov := messenger1.PeerAddresses("unknown peer")
	assert.Equal(t, 0, len(adr1Recov))
}

// ------- ConnectedPeersOnTopic

func TestLibp2pMessenger_ConnectedPeersOnTopicInvalidTopicShouldRetEmptyList(t *testing.T) {
	netw, messenger1, messenger2 := createMockNetworkOf2()
	messenger3, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	defer closeMessengers(messenger1, messenger2, messenger3)

	_ = netw.LinkAll()

	adr2 := messenger2.Addresses()[0]
	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)
	_ = messenger3.ConnectToPeer(adr2)
	// connected peers:  1 ----- 2 ----- 3
	connPeers := messenger1.ConnectedPeersOnTopic("non-existent topic")
	assert.Equal(t, 0, len(connPeers))
}

func TestLibp2pMessenger_ConnectedPeersOnTopicOneTopicShouldWork(t *testing.T) {
	netw, messenger1, messenger2 := createMockNetworkOf2()
	messenger3, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	messenger4, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	defer closeMessengers(messenger1, messenger2, messenger3, messenger4)

	_ = netw.LinkAll()

	adr2 := messenger2.Addresses()[0]
	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)
	_ = messenger3.ConnectToPeer(adr2)
	_ = messenger4.ConnectToPeer(adr2)
	// connected peers:  1 ----- 2 ----- 3
	//                          |
	//                          4
	// 1, 2, 3 should be on topic "topic123"
	_ = messenger1.CreateTopic("topic123", false)
	_ = messenger2.CreateTopic("topic123", false)
	_ = messenger3.CreateTopic("topic123", false)

	// wait a bit for topic announcements
	time.Sleep(time.Second)

	peersOnTopic123 := messenger2.ConnectedPeersOnTopic("topic123")

	assert.Equal(t, 2, len(peersOnTopic123))
	assert.True(t, containsPeerID(peersOnTopic123, messenger1.ID()))
	assert.True(t, containsPeerID(peersOnTopic123, messenger3.ID()))
}

func TestLibp2pMessenger_ConnectedPeersOnTopicOneTopicDifferentViewsShouldWork(t *testing.T) {
	netw, messenger1, messenger2 := createMockNetworkOf2()
	messenger3, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	messenger4, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	defer closeMessengers(messenger1, messenger2, messenger3, messenger4)

	_ = netw.LinkAll()

	adr2 := messenger2.Addresses()[0]
	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)
	_ = messenger3.ConnectToPeer(adr2)
	_ = messenger4.ConnectToPeer(adr2)
	// connected peers:  1 ----- 2 ----- 3
	//                          |
	//                          4
	// 1, 2, 3 should be on topic "topic123"
	_ = messenger1.CreateTopic("topic123", false)
	_ = messenger2.CreateTopic("topic123", false)
	_ = messenger3.CreateTopic("topic123", false)

	// wait a bit for topic announcements
	time.Sleep(time.Second)

	peersOnTopic123FromMessenger2 := messenger2.ConnectedPeersOnTopic("topic123")
	peersOnTopic123FromMessenger4 := messenger4.ConnectedPeersOnTopic("topic123")

	// keep the same checks as the test above as to be 100% that the returned list are correct
	assert.Equal(t, 2, len(peersOnTopic123FromMessenger2))
	assert.True(t, containsPeerID(peersOnTopic123FromMessenger2, messenger1.ID()))
	assert.True(t, containsPeerID(peersOnTopic123FromMessenger2, messenger3.ID()))

	assert.Equal(t, 1, len(peersOnTopic123FromMessenger4))
	assert.True(t, containsPeerID(peersOnTopic123FromMessenger4, messenger2.ID()))
}

func TestLibp2pMessenger_ConnectedPeersOnTopicTwoTopicsShouldWork(t *testing.T) {
	netw, messenger1, messenger2 := createMockNetworkOf2()
	messenger3, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	messenger4, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	defer closeMessengers(messenger1, messenger2, messenger3, messenger4)

	_ = netw.LinkAll()

	adr2 := messenger2.Addresses()[0]
	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)
	_ = messenger3.ConnectToPeer(adr2)
	_ = messenger4.ConnectToPeer(adr2)
	// connected peers:  1 ----- 2 ----- 3
	//                          |
	//                          4
	// 1, 2, 3 should be on topic "topic123"
	// 2, 4 should be on topic "topic24"
	_ = messenger1.CreateTopic("topic123", false)
	_ = messenger2.CreateTopic("topic123", false)
	_ = messenger2.CreateTopic("topic24", false)
	_ = messenger3.CreateTopic("topic123", false)
	_ = messenger4.CreateTopic("topic24", false)

	// wait a bit for topic announcements
	time.Sleep(time.Second)

	peersOnTopic123 := messenger2.ConnectedPeersOnTopic("topic123")
	peersOnTopic24 := messenger2.ConnectedPeersOnTopic("topic24")

	// keep the same checks as the test above as to be 100% that the returned list are correct
	assert.Equal(t, 2, len(peersOnTopic123))
	assert.True(t, containsPeerID(peersOnTopic123, messenger1.ID()))
	assert.True(t, containsPeerID(peersOnTopic123, messenger3.ID()))

	assert.Equal(t, 1, len(peersOnTopic24))
	assert.True(t, containsPeerID(peersOnTopic24, messenger4.ID()))
}

// ------- ConnectedFullHistoryPeersOnTopic

func TestLibp2pMessenger_ConnectedFullHistoryPeersOnTopicShouldWork(t *testing.T) {
	messenger1, messenger2, messenger3 := createMockNetworkOf3()
	defer closeMessengers(messenger1, messenger2, messenger3)

	adr2 := messenger2.Addresses()[0]
	adr3 := messenger3.Addresses()[0]
	fmt.Println("Connecting ...")

	_ = messenger1.ConnectToPeer(adr2)
	_ = messenger3.ConnectToPeer(adr2)
	_ = messenger1.ConnectToPeer(adr3)
	// connected peers:  1 ----- 2
	//                   |       |
	//                   3 ------+

	_ = messenger1.CreateTopic("topic123", false)
	_ = messenger2.CreateTopic("topic123", false)
	_ = messenger3.CreateTopic("topic123", false)

	// wait a bit for topic announcements
	time.Sleep(time.Second)

	assert.Equal(t, 2, len(messenger1.ConnectedPeersOnTopic("topic123")))
	assert.Equal(t, 1, len(messenger1.ConnectedFullHistoryPeersOnTopic("topic123")))

	assert.Equal(t, 2, len(messenger2.ConnectedPeersOnTopic("topic123")))
	assert.Equal(t, 1, len(messenger2.ConnectedFullHistoryPeersOnTopic("topic123")))

	assert.Equal(t, 2, len(messenger3.ConnectedPeersOnTopic("topic123")))
	assert.Equal(t, 2, len(messenger3.ConnectedFullHistoryPeersOnTopic("topic123")))
}

func TestLibp2pMessenger_ConnectedPeersShouldReturnUniquePeers(t *testing.T) {
	pid1 := core.PeerID("pid1")
	pid2 := core.PeerID("pid2")
	pid3 := core.PeerID("pid3")
	pid4 := core.PeerID("pid4")

	hs := &mock.ConnectableHostStub{
		NetworkCalled: func() network.Network {
			return &mock.NetworkStub{
				ConnsCalled: func() []network.Conn {
					// generate a mock list that contain duplicates
					return []network.Conn{
						generateConnWithRemotePeer(pid1),
						generateConnWithRemotePeer(pid1),
						generateConnWithRemotePeer(pid2),
						generateConnWithRemotePeer(pid1),
						generateConnWithRemotePeer(pid4),
						generateConnWithRemotePeer(pid3),
						generateConnWithRemotePeer(pid1),
						generateConnWithRemotePeer(pid3),
						generateConnWithRemotePeer(pid4),
						generateConnWithRemotePeer(pid2),
						generateConnWithRemotePeer(pid1),
						generateConnWithRemotePeer(pid1),
					}
				},
				ConnectednessCalled: func(id peer.ID) network.Connectedness {
					return network.Connected
				},
			}
		},
	}

	netw := mocknet.New()
	messenger, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	// we can safely close the host as the next operations will be done on a mock
	closeMessengers(messenger)

	messenger.SetHost(hs)

	peerList := messenger.ConnectedPeers()

	assert.Equal(t, 4, len(peerList))
	assert.True(t, existInList(peerList, pid1))
	assert.True(t, existInList(peerList, pid2))
	assert.True(t, existInList(peerList, pid3))
	assert.True(t, existInList(peerList, pid4))
}

func existInList(list []core.PeerID, pid core.PeerID) bool {
	for _, p := range list {
		if bytes.Equal(p.Bytes(), pid.Bytes()) {
			return true
		}
	}

	return false
}

func generateConnWithRemotePeer(pid core.PeerID) network.Conn {
	return &mock.ConnStub{
		RemotePeerCalled: func() peer.ID {
			return peer.ID(pid)
		},
	}
}

func TestLibp2pMessenger_SendDirectWithRealMessengersShouldWork(t *testing.T) {
	msg := []byte("test message")

	args := libp2p.ArgsNetworkMessenger{
		Marshalizer:   &mock.ProtoMarshallerMock{},
		ListenAddress: libp2p.TestListenAddrWithIp4AndTcp,
		P2pConfig: config.P2PConfig{
			Node: config.NodeConfig{
				Port: "0",
			},
			KadDhtPeerDiscovery: config.KadDhtPeerDiscoveryConfig{
				Enabled: false,
			},
			Sharding: config.ShardingConfig{
				Type: p2p.NilListSharder,
			},
		},
		SyncTimer:             &libp2p.LocalSyncTimer{},
		PreferredPeersHolder:  &mock.PeersHolderStub{},
		PeersRatingHandler:    &mock.PeersRatingHandlerStub{},
		ConnectionWatcherType: "print",
		P2pPrivateKey:         &mock.PrivateKeyStub{},
		P2pSingleSigner: &mock.SingleSignerStub{
			SignCalled: func(private commonCrypto.PrivateKey, msg []byte) ([]byte, error) {
				return bytes.Repeat([]byte("a"), 70), nil
			},
		},
		P2pKeyGenerator: &mock.KeyGenStub{},
	}
	args.P2pPrivateKey = mock.NewPrivateKeyMock()
	messenger1, _ := libp2p.NewNetworkMessenger(args)
	args.P2pPrivateKey = mock.NewPrivateKeyMock()
	messenger2, _ := libp2p.NewNetworkMessenger(args)
	defer closeMessengers(messenger1, messenger2)

	adr2 := messenger2.Addresses()[0]

	fmt.Printf("Connecting to %s...\n", adr2)

	err := messenger1.ConnectToPeer(adr2)
	require.Nil(t, err)

	wg := &sync.WaitGroup{}
	chanDone := make(chan bool)
	wg.Add(1)

	go func() {
		wg.Wait()
		chanDone <- true
	}()

	minimumSigSize := 70
	err = messenger1.CreateTopic(testTopic, false)
	require.Nil(t, err)
	prepareMessengerForMatchDataReceive(
		messenger2,
		msg,
		wg,
		func(sigSize int) bool {
			return sigSize >= minimumSigSize // message has a non-empty signature
		},
	)

	log.Info("Delaying as to allow peers to announce themselves on the opened topic...")
	time.Sleep(time.Second)

	log.Info("sending message", "from", messenger1.ID().Pretty(), "to", messenger2.ID().Pretty())

	err = messenger1.SendToConnectedPeer("test", msg, messenger2.ID())
	assert.Nil(t, err)

	waitDoneWithTimeout(t, chanDone, timeoutWaitResponses)
}

func TestLibp2pMessenger_SendDirectWithRealMessengersWithoutSignatureShouldWork(t *testing.T) {
	msg := []byte("test message")

	args := libp2p.ArgsNetworkMessenger{
		Marshalizer:   &mock.ProtoMarshallerMock{},
		ListenAddress: libp2p.TestListenAddrWithIp4AndTcp,
		P2pConfig: config.P2PConfig{
			Node: config.NodeConfig{
				Port: "0",
			},
			KadDhtPeerDiscovery: config.KadDhtPeerDiscoveryConfig{
				Enabled: false,
			},
			Sharding: config.ShardingConfig{
				Type: p2p.NilListSharder,
			},
		},
		SyncTimer:             &libp2p.LocalSyncTimer{},
		PreferredPeersHolder:  &mock.PeersHolderStub{},
		PeersRatingHandler:    &mock.PeersRatingHandlerStub{},
		ConnectionWatcherType: "print",
		P2pPrivateKey:         &mock.PrivateKeyStub{},
		P2pSingleSigner:       &mock.SingleSignerStub{},
		P2pKeyGenerator:       &mock.KeyGenStub{},
	}
	args.P2pPrivateKey = mock.NewPrivateKeyMock()
	messenger1, err := libp2p.NewNetworkMessenger(args)
	require.Nil(t, err)
	// force messenger1 not to sign a direct message
	messenger1.SetSignerInDirectSender(&noSigner{messenger1})

	args.P2pPrivateKey = mock.NewPrivateKeyMock()
	messenger2, err := libp2p.NewNetworkMessenger(args)
	require.Nil(t, err)
	defer closeMessengers(messenger1, messenger2)

	adr2 := messenger2.Addresses()[0]

	fmt.Printf("Connecting to %s...\n", adr2)

	_ = messenger1.ConnectToPeer(adr2)

	wg := &sync.WaitGroup{}
	chanDone := make(chan bool)
	wg.Add(1)

	go func() {
		wg.Wait()
		chanDone <- true
	}()

	expectedSigSize := 0
	_ = messenger1.CreateTopic(testTopic, false)
	prepareMessengerForMatchDataReceive(
		messenger2,
		msg,
		wg,
		func(sigSize int) bool {
			return sigSize == expectedSigSize // message an empty signature
		},
	)

	fmt.Println("Delaying as to allow peers to announce themselves on the opened topic...")
	time.Sleep(time.Second)

	fmt.Printf("sending message from %s...\n", messenger1.ID().Pretty())

	err = messenger1.SendToConnectedPeer("test", msg, messenger2.ID())
	assert.Nil(t, err)

	waitDoneWithTimeout(t, chanDone, timeoutWaitResponses)
}

func TestLibp2pMessenger_SendDirectWithRealNetToConnectedPeerShouldWork(t *testing.T) {
	msg := []byte("test message")

	fmt.Println("Messenger 1:")
	messenger1, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())

	fmt.Println("Messenger 2:")
	messenger2, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())
	defer closeMessengers(messenger1, messenger2)

	err := messenger1.ConnectToPeer(getConnectableAddress(messenger2))
	assert.Nil(t, err)

	wg := &sync.WaitGroup{}
	chanDone := make(chan bool)
	wg.Add(2)

	go func() {
		wg.Wait()
		chanDone <- true
	}()

	prepareMessengerForMatchDataReceive(messenger1, msg, wg, noSigCheckHandler)
	prepareMessengerForMatchDataReceive(messenger2, msg, wg, noSigCheckHandler)

	fmt.Println("Delaying as to allow peers to announce themselves on the opened topic...")
	time.Sleep(time.Second)

	fmt.Printf("Messenger 1 is sending message from %s...\n", messenger1.ID().Pretty())
	err = messenger1.SendToConnectedPeer("test", msg, messenger2.ID())
	assert.Nil(t, err)

	time.Sleep(time.Second)
	fmt.Printf("Messenger 2 is sending message from %s...\n", messenger2.ID().Pretty())
	err = messenger2.SendToConnectedPeer("test", msg, messenger1.ID())
	assert.Nil(t, err)

	waitDoneWithTimeout(t, chanDone, timeoutWaitResponses)
}

func TestLibp2pMessenger_SendDirectWithRealNetToSelfShouldWork(t *testing.T) {
	msg := []byte("test message")

	fmt.Println("Messenger 1:")
	messenger, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())
	defer closeMessengers(messenger)

	wg := &sync.WaitGroup{}
	chanDone := make(chan bool)
	wg.Add(1)

	go func() {
		wg.Wait()
		chanDone <- true
	}()

	prepareMessengerForMatchDataReceive(messenger, msg, wg, noSigCheckHandler)

	fmt.Printf("Messenger 1 is sending message from %s to self...\n", messenger.ID().Pretty())
	err := messenger.SendToConnectedPeer("test", msg, messenger.ID())
	assert.Nil(t, err)

	time.Sleep(time.Second)

	waitDoneWithTimeout(t, chanDone, timeoutWaitResponses)
}

// ------- Bootstrap

func TestNetworkMessenger_BootstrapPeerDiscoveryShouldCallPeerBootstrapper(t *testing.T) {
	wasCalled := false

	netw := mocknet.New()
	pdm := &mock.PeerDiscovererStub{
		BootstrapCalled: func() error {
			wasCalled = true
			return nil
		},
		CloseCalled: func() error {
			return nil
		},
	}
	messenger, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	defer closeMessengers(messenger)

	messenger.SetPeerDiscoverer(pdm)

	_ = messenger.Bootstrap()

	assert.True(t, wasCalled)
}

// ------- SetThresholdMinConnectedPeers

func TestNetworkMessenger_SetThresholdMinConnectedPeersInvalidValueShouldErr(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	err := messenger.SetThresholdMinConnectedPeers(-1)

	assert.Equal(t, p2p.ErrInvalidValue, err)
}

func TestNetworkMessenger_SetThresholdMinConnectedPeersShouldWork(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	minConnectedPeers := 56
	err := messenger.SetThresholdMinConnectedPeers(minConnectedPeers)

	assert.Nil(t, err)
	assert.Equal(t, minConnectedPeers, messenger.ThresholdMinConnectedPeers())
}

// ------- IsConnectedToTheNetwork

func TestNetworkMessenger_IsConnectedToTheNetworkRetFalse(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	minConnectedPeers := 56
	_ = messenger.SetThresholdMinConnectedPeers(minConnectedPeers)

	assert.False(t, messenger.IsConnectedToTheNetwork())
}

func TestNetworkMessenger_IsConnectedToTheNetworkWithZeroRetTrue(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	minConnectedPeers := 0
	_ = messenger.SetThresholdMinConnectedPeers(minConnectedPeers)

	assert.True(t, messenger.IsConnectedToTheNetwork())
}

// ------- SetPeerShardResolver

func TestNetworkMessenger_SetPeerShardResolverNilShouldErr(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	err := messenger.SetPeerShardResolver(nil)

	assert.Equal(t, p2p.ErrNilPeerShardResolver, err)
}

func TestNetworkMessenger_SetPeerShardResolver(t *testing.T) {
	messenger := createMockMessenger()
	defer closeMessengers(messenger)

	err := messenger.SetPeerShardResolver(&mock.PeerShardResolverStub{})

	assert.Nil(t, err)
}

func TestNetworkMessenger_DoubleCloseShouldWork(t *testing.T) {
	messenger := createMessenger()

	time.Sleep(time.Second)

	err := messenger.Close()
	assert.Nil(t, err)

	err = messenger.Close()
	assert.Nil(t, err)
}

func TestNetworkMessenger_PreventReprocessingShouldWork(t *testing.T) {
	args := libp2p.ArgsNetworkMessenger{
		ListenAddress: libp2p.TestListenAddrWithIp4AndTcp,
		Marshalizer:   &mock.ProtoMarshallerMock{},
		P2pConfig: config.P2PConfig{
			Node: config.NodeConfig{
				Port: "0",
			},
			KadDhtPeerDiscovery: config.KadDhtPeerDiscoveryConfig{
				Enabled: false,
			},
			Sharding: config.ShardingConfig{
				Type: p2p.NilListSharder,
			},
		},
		SyncTimer:             &libp2p.LocalSyncTimer{},
		PreferredPeersHolder:  &mock.PeersHolderStub{},
		PeersRatingHandler:    &mock.PeersRatingHandlerStub{},
		ConnectionWatcherType: p2p.ConnectionWatcherTypePrint,
		P2pPrivateKey:         mock.NewPrivateKeyMock(),
		P2pSingleSigner:       &mock.SingleSignerStub{},
		P2pKeyGenerator:       &mock.KeyGenStub{},
	}

	messenger, _ := libp2p.NewNetworkMessenger(args)
	defer closeMessengers(messenger)

	numCalled := uint32(0)
	handler := &mock.MessageProcessorStub{
		ProcessMessageCalled: func(message p2p.MessageP2P, fromConnectedPeer core.PeerID) error {
			atomic.AddUint32(&numCalled, 1)
			return nil
		},
	}

	callBackFunc := messenger.PubsubCallback(handler, "")
	ctx := context.Background()
	pid := peer.ID(messenger.ID())
	timeStamp := time.Now().Unix() - 1
	timeStamp -= int64(libp2p.AcceptMessagesInAdvanceDuration.Seconds())
	timeStamp -= int64(libp2p.PubsubTimeCacheDuration.Seconds())

	innerMessage := &data.TopicMessage{
		Payload:   []byte("data"),
		Timestamp: timeStamp,
	}
	buff, _ := args.Marshalizer.Marshal(innerMessage)
	msg := &pubsub.Message{
		Message: &pb.Message{
			From:                 []byte(pid),
			Data:                 buff,
			Seqno:                []byte{0, 0, 0, 1},
			Topic:                nil,
			Signature:            nil,
			Key:                  nil,
			XXX_NoUnkeyedLiteral: struct{}{},
			XXX_unrecognized:     nil,
			XXX_sizecache:        0,
		},
		ReceivedFrom:  "",
		ValidatorData: nil,
	}

	assert.False(t, callBackFunc(ctx, pid, msg)) // this will not call
	assert.False(t, callBackFunc(ctx, pid, msg)) // this will not call
	assert.Equal(t, uint32(0), atomic.LoadUint32(&numCalled))
}

func TestNetworkMessenger_PubsubCallbackNotMessageNotValidShouldNotCallHandler(t *testing.T) {
	args := libp2p.ArgsNetworkMessenger{
		Marshalizer:   &mock.ProtoMarshallerMock{},
		ListenAddress: libp2p.TestListenAddrWithIp4AndTcp,
		P2pConfig: config.P2PConfig{
			Node: config.NodeConfig{
				Port: "0",
			},
			KadDhtPeerDiscovery: config.KadDhtPeerDiscoveryConfig{
				Enabled: false,
			},
			Sharding: config.ShardingConfig{
				Type: p2p.NilListSharder,
			},
		},
		SyncTimer:             &libp2p.LocalSyncTimer{},
		PreferredPeersHolder:  &mock.PeersHolderStub{},
		PeersRatingHandler:    &mock.PeersRatingHandlerStub{},
		ConnectionWatcherType: p2p.ConnectionWatcherTypePrint,
		P2pPrivateKey:         mock.NewPrivateKeyMock(),
		P2pSingleSigner:       &mock.SingleSignerStub{},
		P2pKeyGenerator:       &mock.KeyGenStub{},
	}

	messenger, _ := libp2p.NewNetworkMessenger(args)
	defer closeMessengers(messenger)

	numUpserts := int32(0)
	_ = messenger.SetPeerDenialEvaluator(&mock.PeerDenialEvaluatorStub{
		UpsertPeerIDCalled: func(pid core.PeerID, duration time.Duration) error {
			atomic.AddInt32(&numUpserts, 1)
			// any error thrown here should not impact the execution
			return fmt.Errorf("expected error")
		},
		IsDeniedCalled: func(pid core.PeerID) bool {
			return false
		},
	})

	numCalled := uint32(0)
	handler := &mock.MessageProcessorStub{
		ProcessMessageCalled: func(message p2p.MessageP2P, fromConnectedPeer core.PeerID) error {
			atomic.AddUint32(&numCalled, 1)
			return nil
		},
	}

	callBackFunc := messenger.PubsubCallback(handler, "")
	ctx := context.Background()
	pid := peer.ID(messenger.ID())
	innerMessage := &data.TopicMessage{
		Payload:   []byte("data"),
		Timestamp: time.Now().Unix(),
	}
	buff, _ := args.Marshalizer.Marshal(innerMessage)
	msg := &pubsub.Message{
		Message: &pb.Message{
			From:                 []byte("not a valid pid"),
			Data:                 buff,
			Seqno:                []byte{0, 0, 0, 1},
			Topic:                nil,
			Signature:            nil,
			Key:                  nil,
			XXX_NoUnkeyedLiteral: struct{}{},
			XXX_unrecognized:     nil,
			XXX_sizecache:        0,
		},
		ReceivedFrom:  "",
		ValidatorData: nil,
	}

	assert.False(t, callBackFunc(ctx, pid, msg))
	assert.Equal(t, uint32(0), atomic.LoadUint32(&numCalled))
	assert.Equal(t, int32(2), atomic.LoadInt32(&numUpserts))
}

func TestNetworkMessenger_PubsubCallbackReturnsFalseIfHandlerErrors(t *testing.T) {
	args := libp2p.ArgsNetworkMessenger{
		Marshalizer:   &mock.ProtoMarshallerMock{},
		ListenAddress: libp2p.TestListenAddrWithIp4AndTcp,
		P2pConfig: config.P2PConfig{
			Node: config.NodeConfig{
				Port: "0",
			},
			KadDhtPeerDiscovery: config.KadDhtPeerDiscoveryConfig{
				Enabled: false,
			},
			Sharding: config.ShardingConfig{
				Type: p2p.NilListSharder,
			},
		},
		SyncTimer:             &libp2p.LocalSyncTimer{},
		PreferredPeersHolder:  &mock.PeersHolderStub{},
		PeersRatingHandler:    &mock.PeersRatingHandlerStub{},
		ConnectionWatcherType: p2p.ConnectionWatcherTypePrint,
		P2pPrivateKey:         mock.NewPrivateKeyMock(),
		P2pSingleSigner:       &mock.SingleSignerStub{},
		P2pKeyGenerator:       &mock.KeyGenStub{},
	}

	messenger, _ := libp2p.NewNetworkMessenger(args)
	defer closeMessengers(messenger)

	numCalled := uint32(0)
	expectedErr := errors.New("expected error")
	handler := &mock.MessageProcessorStub{
		ProcessMessageCalled: func(message p2p.MessageP2P, fromConnectedPeer core.PeerID) error {
			atomic.AddUint32(&numCalled, 1)
			return expectedErr
		},
	}

	callBackFunc := messenger.PubsubCallback(handler, "")
	ctx := context.Background()
	pid := peer.ID(messenger.ID())
	innerMessage := &data.TopicMessage{
		Payload:   []byte("data"),
		Timestamp: time.Now().Unix(),
		Version:   libp2p.CurrentTopicMessageVersion,
	}
	buff, _ := args.Marshalizer.Marshal(innerMessage)
	topic := "topic"
	msg := &pubsub.Message{
		Message: &pb.Message{
			From:                 []byte(messenger.ID()),
			Data:                 buff,
			Seqno:                []byte{0, 0, 0, 1},
			Topic:                &topic,
			Signature:            nil,
			Key:                  nil,
			XXX_NoUnkeyedLiteral: struct{}{},
			XXX_unrecognized:     nil,
			XXX_sizecache:        0,
		},
		ReceivedFrom:  "",
		ValidatorData: nil,
	}

	assert.False(t, callBackFunc(ctx, pid, msg))
	assert.Equal(t, uint32(1), atomic.LoadUint32(&numCalled))
}

func TestNetworkMessenger_UnjoinAllTopicsShouldWork(t *testing.T) {
	args := libp2p.ArgsNetworkMessenger{
		Marshalizer:   &mock.ProtoMarshallerMock{},
		ListenAddress: libp2p.TestListenAddrWithIp4AndTcp,
		P2pConfig: config.P2PConfig{
			Node: config.NodeConfig{
				Port: "0",
			},
			KadDhtPeerDiscovery: config.KadDhtPeerDiscoveryConfig{
				Enabled: false,
			},
			Sharding: config.ShardingConfig{
				Type: p2p.NilListSharder,
			},
		},
		SyncTimer:             &libp2p.LocalSyncTimer{},
		PreferredPeersHolder:  &mock.PeersHolderStub{},
		PeersRatingHandler:    &mock.PeersRatingHandlerStub{},
		ConnectionWatcherType: p2p.ConnectionWatcherTypePrint,
		P2pPrivateKey:         mock.NewPrivateKeyMock(),
		P2pSingleSigner:       &mock.SingleSignerStub{},
		P2pKeyGenerator:       &mock.KeyGenStub{},
	}

	messenger, _ := libp2p.NewNetworkMessenger(args)
	defer closeMessengers(messenger)

	topic := "topic"
	_ = messenger.CreateTopic(topic, true)
	assert.True(t, messenger.HasTopic(topic))

	err := messenger.UnjoinAllTopics()
	assert.Nil(t, err)

	assert.False(t, messenger.HasTopic(topic))
}

func TestNetworkMessenger_ValidMessageByTimestampMessageTooOld(t *testing.T) {
	args := createMockNetworkArgs()
	now := time.Now()
	args.SyncTimer = &mock.SyncTimerStub{
		CurrentTimeCalled: func() time.Time {
			return now
		},
	}
	messenger, _ := libp2p.NewNetworkMessenger(args)
	defer closeMessengers(messenger)

	msg := &message.Message{
		TimestampField: now.Unix() - int64(libp2p.PubsubTimeCacheDuration.Seconds()) - 1,
	}
	err := messenger.ValidMessageByTimestamp(msg)

	assert.True(t, errors.Is(err, p2p.ErrMessageTooOld))
}

func TestNetworkMessenger_ValidMessageByTimestampMessageAtLowerLimitShouldWork(t *testing.T) {
	messenger, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())
	defer closeMessengers(messenger)

	now := time.Now()
	msg := &message.Message{
		TimestampField: now.Unix() - int64(libp2p.PubsubTimeCacheDuration.Seconds()) + int64(libp2p.AcceptMessagesInAdvanceDuration.Seconds()),
	}
	err := messenger.ValidMessageByTimestamp(msg)

	assert.Nil(t, err)
}

func TestNetworkMessenger_ValidMessageByTimestampMessageTooNew(t *testing.T) {
	args := createMockNetworkArgs()
	now := time.Now()
	args.SyncTimer = &mock.SyncTimerStub{
		CurrentTimeCalled: func() time.Time {
			return now
		},
	}
	messenger, _ := libp2p.NewNetworkMessenger(args)
	defer closeMessengers(messenger)

	msg := &message.Message{
		TimestampField: now.Unix() + int64(libp2p.AcceptMessagesInAdvanceDuration.Seconds()) + 1,
	}
	err := messenger.ValidMessageByTimestamp(msg)

	assert.True(t, errors.Is(err, p2p.ErrMessageTooNew))
}

func TestNetworkMessenger_ValidMessageByTimestampMessageAtUpperLimitShouldWork(t *testing.T) {
	args := createMockNetworkArgs()
	now := time.Now()
	args.SyncTimer = &mock.SyncTimerStub{
		CurrentTimeCalled: func() time.Time {
			return now
		},
	}
	messenger, _ := libp2p.NewNetworkMessenger(args)
	defer closeMessengers(messenger)

	msg := &message.Message{
		TimestampField: now.Unix() + int64(libp2p.AcceptMessagesInAdvanceDuration.Seconds()),
	}
	err := messenger.ValidMessageByTimestamp(msg)

	assert.Nil(t, err)
}

func TestNetworkMessenger_GetConnectedPeersInfo(t *testing.T) {
	netw := mocknet.New()

	peers := []peer.ID{
		"valI1",
		"valC1",
		"valC2",
		"obsI1",
		"obsI2",
		"obsI3",
		"obsC1",
		"obsC2",
		"obsC3",
		"obsC4",
		"unknown",
	}
	messenger, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
	closeMessengers(messenger)

	messenger.SetHost(&mock.ConnectableHostStub{
		NetworkCalled: func() network.Network {
			return &mock.NetworkStub{
				PeersCall: func() []peer.ID {
					return peers
				},
				ConnsToPeerCalled: func(p peer.ID) []network.Conn {
					return make([]network.Conn, 0)
				},
			}
		},
	})
	selfShardID := uint32(0)
	crossShardID := uint32(1)
	_ = messenger.SetPeerShardResolver(&mock.PeerShardResolverStub{
		GetPeerInfoCalled: func(pid core.PeerID) core.P2PPeerInfo {
			pinfo := core.P2PPeerInfo{
				PeerType: core.UnknownPeer,
			}
			if pid.Pretty() == messenger.ID().Pretty() {
				pinfo.ShardID = selfShardID
				pinfo.PeerType = core.ObserverPeer
				return pinfo
			}

			strPid := string(pid)
			if strings.Contains(strPid, "I") {
				pinfo.ShardID = selfShardID
			}
			if strings.Contains(strPid, "C") {
				pinfo.ShardID = crossShardID
			}

			if strings.Contains(strPid, "val") {
				pinfo.PeerType = core.ValidatorPeer
			}

			if strings.Contains(strPid, "obs") {
				pinfo.PeerType = core.ObserverPeer
			}

			return pinfo
		},
	})

	cpi := messenger.GetConnectedPeersInfo()

	assert.Equal(t, 4, cpi.NumCrossShardObservers)
	assert.Equal(t, 2, cpi.NumCrossShardValidators)
	assert.Equal(t, 3, cpi.NumIntraShardObservers)
	assert.Equal(t, 1, cpi.NumIntraShardValidators)
	assert.Equal(t, 3, cpi.NumObserversOnShard[selfShardID])
	assert.Equal(t, 4, cpi.NumObserversOnShard[crossShardID])
	assert.Equal(t, 1, cpi.NumValidatorsOnShard[selfShardID])
	assert.Equal(t, 2, cpi.NumValidatorsOnShard[crossShardID])
	assert.Equal(t, selfShardID, cpi.SelfShardID)
	assert.Equal(t, 1, len(cpi.UnknownPeers))
}

func TestNetworkMessenger_mapHistogram(t *testing.T) {
	args := createMockNetworkArgs()
	messenger, _ := libp2p.NewNetworkMessenger(args)
	defer closeMessengers(messenger)

	inp := map[uint32]int{
		0:                     5,
		1:                     7,
		2:                     9,
		core.MetachainShardId: 11,
	}
	output := `shard 0: 5, shard 1: 7, shard 2: 9, meta: 11`

	require.Equal(t, output, messenger.MapHistogram(inp))
}

func TestNetworkMessenger_Bootstrap(t *testing.T) {
	t.Skip("long test used to debug go routines closing on the netMessenger")

	_ = logger.SetLogLevel("*:DEBUG")

	args := libp2p.ArgsNetworkMessenger{
		ListenAddress: libp2p.TestListenAddrWithIp4AndTcp,
		Marshalizer:   &marshal.GogoProtoMarshalizer{},
		P2pConfig: config.P2PConfig{
			Node: config.NodeConfig{
				Port:                       "0",
				MaximumExpectedPeerCount:   1,
				ThresholdMinConnectedPeers: 1,
			},
			KadDhtPeerDiscovery: config.KadDhtPeerDiscoveryConfig{
				Enabled:                          true,
				Type:                             "optimized",
				RefreshIntervalInSec:             10,
				ProtocolID:                       "erd/kad/1.0.0",
				InitialPeerList:                  []string{"/ip4/35.214.140.83/tcp/10000/p2p/16Uiu2HAm6hPymvkZyFgbvWaVBKhEoPjmXhkV32r9JaFvQ7Rk8ynU"},
				BucketSize:                       10,
				RoutingTableRefreshIntervalInSec: 5,
			},
			Sharding: config.ShardingConfig{
				TargetPeerCount:         0,
				MaxIntraShardValidators: 0,
				MaxCrossShardValidators: 0,
				MaxIntraShardObservers:  0,
				MaxCrossShardObservers:  0,
				MaxSeeders:              0,
				Type:                    "NilListSharder",
			},
		},
		SyncTimer:            &mock.SyncTimerStub{},
		PeersRatingHandler:   &mock.PeersRatingHandlerStub{},
		PreferredPeersHolder: &mock.PeersHolderStub{}, ConnectionWatcherType: p2p.ConnectionWatcherTypePrint,
		P2pPrivateKey:   mock.NewPrivateKeyMock(),
		P2pSingleSigner: &mock.SingleSignerStub{},
		P2pKeyGenerator: &mock.KeyGenStub{},
	}

	messenger, err := libp2p.NewNetworkMessenger(args)
	require.Nil(t, err)

	go func() {
		time.Sleep(time.Second * 1)
		goRoutinesNumberStart := runtime.NumGoroutine()
		log.Info("before closing", "num go routines", goRoutinesNumberStart)

		closeMessengers(messenger)
	}()

	_ = messenger.Bootstrap()

	time.Sleep(time.Second * 5)

	goRoutinesNumberStart := runtime.NumGoroutine()
	core.DumpGoRoutinesToLog(goRoutinesNumberStart, log)
}

func TestNetworkMessenger_WaitForConnections(t *testing.T) {
	t.Run("min num of peers is 0", func(t *testing.T) {
		startTime := time.Now()
		_, messenger1, messenger2 := createMockNetworkOf2()
		_ = messenger1.ConnectToPeer(messenger2.Addresses()[0])
		defer closeMessengers(messenger1, messenger2)

		timeToWait := time.Second * 3
		messenger1.WaitForConnections(timeToWait, 0)

		assert.True(t, timeToWait <= time.Since(startTime))
	})
	t.Run("min num of peers is 2", func(t *testing.T) {
		startTime := time.Now()
		netw, messenger1, messenger2 := createMockNetworkOf2()
		messenger3, _ := libp2p.NewMockMessenger(createMockNetworkArgs(), netw)
		defer closeMessengers(messenger1, messenger2, messenger3)
		_ = netw.LinkAll()

		_ = messenger1.ConnectToPeer(messenger2.Addresses()[0])
		go func() {
			time.Sleep(time.Second * 2)
			_ = messenger1.ConnectToPeer(messenger3.Addresses()[0])
		}()

		timeToWait := time.Second * 10
		messenger1.WaitForConnections(timeToWait, 2)

		assert.True(t, timeToWait > time.Since(startTime))
		assert.True(t, libp2p.PollWaitForConnectionsInterval <= time.Since(startTime))
	})
	t.Run("min num of peers is 2 but we only connected to 1 peer", func(t *testing.T) {
		startTime := time.Now()
		_, messenger1, messenger2 := createMockNetworkOf2()
		defer closeMessengers(messenger1, messenger2)

		_ = messenger1.ConnectToPeer(messenger2.Addresses()[0])

		timeToWait := time.Second * 10
		messenger1.WaitForConnections(timeToWait, 2)

		assert.True(t, timeToWait < time.Since(startTime))
	})
}

func TestLibp2pMessenger_SignVerifyPayloadShouldWork(t *testing.T) {
	fmt.Println("Messenger 1:")
	messenger1, err := libp2p.NewNetworkMessenger(createMockNetworkArgs())
	require.Nil(t, err)

	fmt.Println("Messenger 2:")
	messenger2, err := libp2p.NewNetworkMessenger(createMockNetworkArgs())
	require.Nil(t, err)

	defer closeMessengers(messenger1, messenger2)

	err = messenger1.ConnectToPeer(getConnectableAddress(messenger2))
	assert.Nil(t, err)

	payload := []byte("payload")
	sig, err := messenger1.Sign(payload)
	assert.Nil(t, err)

	err = messenger2.Verify(payload, messenger1.ID(), sig)
	assert.Nil(t, err)

	err = messenger1.Verify(payload, messenger1.ID(), sig)
	assert.Nil(t, err)
}

func TestNetworkMessenger_BroadcastUsingPrivateKey(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	msg := []byte("test message")
	topic := "topic"

	interceptors := make([]*mock.MessageProcessorMock, 2)

	fmt.Println("Messenger 1:")
	messenger1, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())
	_ = messenger1.CreateTopic(topic, true)
	interceptors[0] = mock.NewMessageProcessorMock()
	_ = messenger1.RegisterMessageProcessor(topic, "", interceptors[0])

	fmt.Println("Messenger 2:")
	messenger2, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())
	_ = messenger2.CreateTopic(topic, true)
	interceptors[1] = mock.NewMessageProcessorMock()
	_ = messenger2.RegisterMessageProcessor(topic, "", interceptors[1])

	defer closeMessengers(messenger1, messenger2)

	err := messenger1.ConnectToPeer(getConnectableAddress(messenger2))
	assert.Nil(t, err)

	time.Sleep(time.Second * 2)

	skBuff, peerID := createP2PPrivKeyAndPid()
	pid := core.PeerID(peerID)
	fmt.Printf("new identity: %s\n", pid.Pretty())

	messenger1.BroadcastUsingPrivateKey(topic, msg, pid, skBuff)

	time.Sleep(time.Second * 2)

	for _, i := range interceptors {
		messages := i.GetMessages()

		assert.Equal(t, 1, len(messages))
		assert.Equal(t, 1, messages[pid])
		assert.Equal(t, 0, messages[messenger1.ID()])
		assert.Equal(t, 0, messages[messenger2.ID()])
	}
}

func createP2PPrivKeyAndPid() ([]byte, peer.ID) {
	keyGen := signing.NewKeyGenerator(secp256k1.NewSecp256k1())
	prvKey, _ := keyGen.GeneratePair()

	p2pPrivKey, _ := p2pCrypto.ConvertPrivateKeyToLibp2pPrivateKey(prvKey)
	p2pPubKey := p2pPrivKey.GetPublic()
	pid, _ := peer.IDFromPublicKey(p2pPubKey)

	p2pPrivKeyBytes, _ := p2pPrivKey.Raw()

	return p2pPrivKeyBytes, pid
}

func TestNetworkMessenger_AddPeerTopicNotifier(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	waitForPubSubTime := time.Second * 3

	t.Run("nil topic notifier should error", func(t *testing.T) {
		messenger, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())
		defer closeMessengers(messenger)

		err := messenger.AddPeerTopicNotifier(nil)
		assert.Equal(t, p2p.ErrNilPeerTopicNotifier, err)
	})
	t.Run("2 peers on different topics should not notify", func(t *testing.T) {
		messenger1, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())
		messenger2, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())
		defer closeMessengers(messenger1, messenger2)

		peerTopicNotifier := &mock.PeerTopicNotifierStub{
			NewPeerFoundCalled: func(pid core.PeerID, topic string) {
				assert.Fail(t, fmt.Sprintf("should have not notified: topic %s, pid %s", topic, pid.Pretty()))
			},
		}

		err := messenger1.AddPeerTopicNotifier(peerTopicNotifier)
		assert.Nil(t, err)
		err = messenger2.AddPeerTopicNotifier(peerTopicNotifier)
		assert.Nil(t, err)

		_ = messenger1.ConnectToPeer(messenger2.Addresses()[0])
		time.Sleep(waitForPubSubTime) // wait a bit for pubsub

		_ = messenger1.CreateTopic("topic1", true)
		_ = messenger1.CreateTopic("topic2", true)

		time.Sleep(waitForPubSubTime) // wait a bit for pubsub
	})
	t.Run("2 peers on same topic should notify", func(t *testing.T) {
		messenger1, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())
		messenger2, _ := libp2p.NewNetworkMessenger(createMockNetworkArgs())
		defer closeMessengers(messenger1, messenger2)

		mut := sync.RWMutex{}
		peersOnTopicsFound := make(map[core.PeerID]map[string]int)

		peerTopicNotifier := &mock.PeerTopicNotifierStub{
			NewPeerFoundCalled: func(pid core.PeerID, topic string) {
				mut.Lock()
				topics := peersOnTopicsFound[pid]
				if topics == nil {
					topics = make(map[string]int)
					peersOnTopicsFound[pid] = topics
				}

				// warning: pubsub v0.8.1 might call multiple times the handler defined in pubsub.WithPeerFilter call for a new peer found
				peersOnTopicsFound[pid][topic]++
				mut.Unlock()
			},
		}

		err := messenger1.AddPeerTopicNotifier(peerTopicNotifier)
		assert.Nil(t, err)
		err = messenger2.AddPeerTopicNotifier(peerTopicNotifier)
		assert.Nil(t, err)

		_ = messenger1.ConnectToPeer(messenger2.Addresses()[0])
		log.Info("netMes1 connected to netMes2, waiting on pubsub")
		time.Sleep(waitForPubSubTime)

		mut.RLock()
		assert.Equal(t, 0, len(peersOnTopicsFound))
		mut.RUnlock()

		log.Info("creating topic1 on netMes1 and netMes2 an then waiting on pubsub")
		_ = messenger1.CreateTopic("topic1", true)
		_ = messenger2.CreateTopic("topic1", true)
		time.Sleep(waitForPubSubTime)

		mut.RLock()
		assert.Equal(t, 2, len(peersOnTopicsFound))
		assert.True(t, peersOnTopicsFound[messenger1.ID()]["topic1"] >= 1)
		assert.True(t, peersOnTopicsFound[messenger2.ID()]["topic1"] >= 1)
		mut.RUnlock()

		log.Info("creating topic2 on netMes1 and netMes2 an then waiting on pubsub")
		_ = messenger1.CreateTopic("topic2", true)
		_ = messenger2.CreateTopic("topic2", true)
		time.Sleep(waitForPubSubTime)

		mut.RLock()
		assert.Equal(t, 2, len(peersOnTopicsFound))
		assert.True(t, peersOnTopicsFound[messenger1.ID()]["topic1"] >= 1)
		assert.True(t, peersOnTopicsFound[messenger2.ID()]["topic1"] >= 1)
		assert.True(t, peersOnTopicsFound[messenger1.ID()]["topic2"] >= 1)
		assert.True(t, peersOnTopicsFound[messenger2.ID()]["topic2"] >= 1)
		mut.RUnlock()

		log.Info("disconnecting netMes2 from netMes1...")
		_ = messenger2.Disconnect(messenger1.ID())
		time.Sleep(waitForPubSubTime)

		log.Info("reconnecting netMes2 to netMes1...")
		_ = messenger2.ConnectToPeer(messenger1.Addresses()[0])
		time.Sleep(waitForPubSubTime)

		mut.RLock()
		assert.Equal(t, 2, len(peersOnTopicsFound))
		assert.True(t, peersOnTopicsFound[messenger1.ID()]["topic1"] >= 2)
		assert.True(t, peersOnTopicsFound[messenger2.ID()]["topic1"] >= 2)
		assert.True(t, peersOnTopicsFound[messenger1.ID()]["topic2"] >= 2)
		assert.True(t, peersOnTopicsFound[messenger2.ID()]["topic2"] >= 2)
		mut.RUnlock()
	})
}
