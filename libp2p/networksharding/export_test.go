package networksharding

import (
	"math/big"

	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/libp2p/go-libp2p/core/peer"
)

const MinAllowedConnectedPeersListSharder = minAllowedConnectedPeersListSharder
const MinAllowedValidators = minAllowedValidators
const MinAllowedObservers = minAllowedObservers
const MinAllowedConnectedPeersOneSharder = minAllowedConnectedPeersOneSharder

func (ls *listsSharder) GetMaxPeerCount() int {
	return ls.maxPeerCount
}

func (ls *listsSharder) GetMaxIntraShardValidators() int {
	return ls.maxIntraShardValidators
}

func (ls *listsSharder) GetMaxCrossShardValidators() int {
	return ls.maxCrossShardValidators
}

func (ls *listsSharder) GetMaxIntraShardObservers() int {
	return ls.maxIntraShardObservers
}

func (ls *listsSharder) GetMaxCrossShardObservers() int {
	return ls.maxCrossShardObservers
}

func (ls *listsSharder) GetMaxSeeders() int {
	return ls.maxSeeders
}

func (ls *listsSharder) GetMaxFullHistoryObservers() int {
	return ls.maxFullHistoryObservers
}

func (ls *listsSharder) GetMaxUnknown() int {
	return ls.maxUnknown
}

func ComputeDistanceByCountingBits(src peer.ID, dest peer.ID) *big.Int {
	return computeDistanceByCountingBits(src, dest)
}

func ComputeDistanceLog2Based(src peer.ID, dest peer.ID) *big.Int {
	return computeDistanceLog2Based(src, dest)
}

func (ls *listsSharder) GetPeerShardResolver() p2p.PeerShardResolver {
	return ls.peerShardResolver
}
