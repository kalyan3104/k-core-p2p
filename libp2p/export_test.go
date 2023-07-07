package libp2p

import (
	"context"
	"time"

	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/kalyan3104/k-core/core"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/kalyan3104/k-core-storage-go/types"
	"github.com/whyrusleeping/timecache"
)

var MaxSendBuffSize = maxSendBuffSize
var BroadcastGoRoutines = broadcastGoRoutines
var PubsubTimeCacheDuration = pubsubTimeCacheDuration
var AcceptMessagesInAdvanceDuration = acceptMessagesInAdvanceDuration
var SequenceNumberSize = sequenceNumberSize

const CurrentTopicMessageVersion = currentTopicMessageVersion
const PollWaitForConnectionsInterval = pollWaitForConnectionsInterval

// SetHost -
func (netMes *networkMessenger) SetHost(newHost ConnectableHost) {
	netMes.p2pHost = newHost
}

// SetLoadBalancer -
func (netMes *networkMessenger) SetLoadBalancer(outgoingPLB ChannelLoadBalancer) {
	netMes.outgoingPLB = outgoingPLB
}

// SetPeerDiscoverer -
func (netMes *networkMessenger) SetPeerDiscoverer(discoverer p2p.PeerDiscoverer) {
	netMes.peerDiscoverer = discoverer
}

// PubsubCallback -
func (netMes *networkMessenger) PubsubCallback(handler p2p.MessageProcessor, topic string) func(ctx context.Context, pid peer.ID, message *pubsub.Message) bool {
	topicProcs := newTopicProcessors()
	_ = topicProcs.addTopicProcessor("identifier", handler)

	return netMes.pubsubCallback(topicProcs, topic)
}

// ValidMessageByTimestamp -
func (netMes *networkMessenger) ValidMessageByTimestamp(msg p2p.MessageP2P) error {
	return netMes.validMessageByTimestamp(msg)
}

// MapHistogram -
func (netMes *networkMessenger) MapHistogram(input map[uint32]int) string {
	return netMes.mapHistogram(input)
}

// PubsubHasTopic -
func (netMes *networkMessenger) PubsubHasTopic(expectedTopic string) bool {
	netMes.mutTopics.RLock()
	topics := netMes.pb.GetTopics()
	netMes.mutTopics.RUnlock()

	for _, topic := range topics {
		if topic == expectedTopic {
			return true
		}
	}
	return false
}

// HasProcessorForTopic -
func (netMes *networkMessenger) HasProcessorForTopic(expectedTopic string) bool {
	processor, found := netMes.processors[expectedTopic]

	return found && processor != nil
}

// Disconnect -
func (netMes *networkMessenger) Disconnect(pid core.PeerID) error {
	return netMes.p2pHost.Network().ClosePeer(peer.ID(pid))
}

// ProcessReceivedDirectMessage -
func (ds *directSender) ProcessReceivedDirectMessage(message *pb.Message, fromConnectedPeer peer.ID) error {
	return ds.processReceivedDirectMessage(message, fromConnectedPeer)
}

// SeenMessages -
func (ds *directSender) SeenMessages() *timecache.TimeCache {
	return ds.seenMessages
}

// Counter -
func (ds *directSender) Counter() uint64 {
	return ds.counter
}

// Mutexes -
func (mh *MutexHolder) Mutexes() types.Cacher {
	return mh.mutexes
}

// SetSignerInDirectSender sets the signer in the direct sender
func (netMes *networkMessenger) SetSignerInDirectSender(signer p2p.SignerVerifier) {
	netMes.ds.(*directSender).signer = signer
}

func (oplb *OutgoingChannelLoadBalancer) Chans() []chan *SendableData {
	return oplb.chans
}

func (oplb *OutgoingChannelLoadBalancer) Names() []string {
	return oplb.names
}

func (oplb *OutgoingChannelLoadBalancer) NamesChans() map[string]chan *SendableData {
	return oplb.namesChans
}

func DefaultSendChannel() string {
	return defaultSendChannel
}

func NewConnectionMonitorWrapper(
	network network.Network,
	connMonitor ConnectionMonitor,
	peerDenialEvaluator p2p.PeerDenialEvaluator,
) *connectionMonitorWrapper {
	return newConnectionMonitorWrapper(network, connMonitor, peerDenialEvaluator)
}

func NewPeersOnChannel(
	peersRatingHandler p2p.PeersRatingHandler,
	fetchPeersHandler func(topic string) []peer.ID,
	refreshInterval time.Duration,
	ttlInterval time.Duration,
) (*peersOnChannel, error) {
	return newPeersOnChannel(peersRatingHandler, fetchPeersHandler, refreshInterval, ttlInterval)
}

func (poc *peersOnChannel) SetPeersOnTopic(topic string, lastUpdated time.Time, peers []core.PeerID) {
	poc.mutPeers.Lock()
	poc.peers[topic] = peers
	poc.lastUpdated[topic] = lastUpdated
	poc.mutPeers.Unlock()
}

func (poc *peersOnChannel) GetPeers(topic string) []core.PeerID {
	poc.mutPeers.RLock()
	defer poc.mutPeers.RUnlock()

	return poc.peers[topic]
}

func (poc *peersOnChannel) SetTimeHandler(handler func() time.Time) {
	poc.getTimeHandler = handler
}

func GetPort(port string, handler func(int) error) (int, error) {
	return getPort(port, handler)
}

func CheckFreePort(port int) error {
	return checkFreePort(port)
}

func NewTopicProcessors() *topicProcessors {
	return newTopicProcessors()
}

func (tp *topicProcessors) AddTopicProcessor(identifier string, processor p2p.MessageProcessor) error {
	return tp.addTopicProcessor(identifier, processor)
}

func (tp *topicProcessors) RemoveTopicProcessor(identifier string) error {
	return tp.removeTopicProcessor(identifier)
}

func (tp *topicProcessors) GetList() ([]string, []p2p.MessageProcessor) {
	return tp.getList()
}

func NewUnknownPeerShardResolver() *unknownPeerShardResolver {
	return &unknownPeerShardResolver{}
}
