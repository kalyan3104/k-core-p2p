package networksharding_test

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kalyan3104/k-core-p2p/config"
	"github.com/kalyan3104/k-core-p2p/libp2p/networksharding"
	"github.com/kalyan3104/k-core-p2p/mock"
	"github.com/kalyan3104/k-core-p2p/peersHolder" 
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const crtShardId = uint32(0)
const crossShardId = uint32(1)

const validatorMarker = "validator"
const observerMarker = "observer"
const unknownMarker = "unknown"
const seederMarker = "seeder"

var crtPid = peer.ID(fmt.Sprintf("%d pid", crtShardId))

func createStringPeersShardResolver() *mock.PeerShardResolverStub {
	return &mock.PeerShardResolverStub{
		GetPeerInfoCalled: func(pid core.PeerID) core.P2PPeerInfo {
			strPid := string(pid)
			pInfo := core.P2PPeerInfo{}

			if strings.Contains(strPid, fmt.Sprintf("%d", crtShardId)) {
				pInfo.ShardID = crtShardId
			} else {
				pInfo.ShardID = crossShardId
			}

			if strings.Contains(strPid, unknownMarker) {
				pInfo.PeerType = core.UnknownPeer
			}
			if strings.Contains(strPid, validatorMarker) {
				pInfo.PeerType = core.ValidatorPeer
			}
			if strings.Contains(strPid, observerMarker) {
				pInfo.PeerType = core.ObserverPeer
			}

			return pInfo
		},
	}
}

func countPeers(peers []peer.ID, shardID uint32, marker string) int {
	counter := 0
	for _, pid := range peers {
		if strings.Contains(string(pid), marker) &&
			strings.Contains(string(pid), fmt.Sprintf("%d", shardID)) {
			counter++
		}
	}

	return counter
}

func createMockListSharderArguments() networksharding.ArgListsSharder {
	return networksharding.ArgListsSharder{
		PeerResolver:         createStringPeersShardResolver(),
		SelfPeerId:           crtPid,
		PreferredPeersHolder: &mock.PeersHolderStub{},
		P2pConfig: config.P2PConfig{
			Sharding: config.ShardingConfig{
				TargetPeerCount:         networksharding.MinAllowedConnectedPeersListSharder,
				MaxIntraShardValidators: networksharding.MinAllowedValidators,
				MaxCrossShardValidators: networksharding.MinAllowedValidators,
				MaxIntraShardObservers:  networksharding.MinAllowedObservers,
				MaxCrossShardObservers:  networksharding.MinAllowedObservers,
				MaxSeeders:              0,
			},
		},
	}
}

func TestNewListsSharder_InvalidMinimumTargetPeerCountShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.P2pConfig.Sharding.TargetPeerCount = networksharding.MinAllowedConnectedPeersListSharder - 1
	ls, err := networksharding.NewListsSharder(arg)

	assert.True(t, check.IfNil(ls))
	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
	assert.True(t, strings.Contains(err.Error(), "maxPeerCount should be at least"))
}

func TestNewListsSharder_NilPeerShardResolverShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.PeerResolver = nil
	ls, err := networksharding.NewListsSharder(arg)

	assert.True(t, check.IfNil(ls))
	assert.True(t, errors.Is(err, p2p.ErrNilPeerShardResolver))
}

func TestNewListsSharder_InvalidIntraShardValidatorsShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.P2pConfig.Sharding.MaxIntraShardValidators = networksharding.MinAllowedValidators - 1
	ls, err := networksharding.NewListsSharder(arg)

	assert.True(t, check.IfNil(ls))
	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
}

func TestNewListsSharder_InvalidCrossShardValidatorsShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.P2pConfig.Sharding.MaxCrossShardValidators = networksharding.MinAllowedValidators - 1
	ls, err := networksharding.NewListsSharder(arg)

	assert.True(t, check.IfNil(ls))
	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
}

func TestNewListsSharder_InvalidIntraShardObserversShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.P2pConfig.Sharding.MaxIntraShardObservers = networksharding.MinAllowedObservers - 1
	ls, err := networksharding.NewListsSharder(arg)

	assert.True(t, check.IfNil(ls))
	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
}

func TestNewListsSharder_InvalidCrossShardObserversShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.P2pConfig.Sharding.MaxCrossShardObservers = networksharding.MinAllowedObservers - 1
	ls, err := networksharding.NewListsSharder(arg)

	assert.True(t, check.IfNil(ls))
	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
}

func TestNewListsSharder_NoRoomForUnknownShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.P2pConfig.Sharding.MaxCrossShardObservers = networksharding.MinAllowedObservers + 1
	ls, err := networksharding.NewListsSharder(arg)

	assert.True(t, check.IfNil(ls))
	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
}

func TestNewListsSharder_NilPreferredPeersShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.PreferredPeersHolder = nil
	ls, err := networksharding.NewListsSharder(arg)

	assert.True(t, check.IfNil(ls))
	assert.True(t, errors.Is(err, p2p.ErrNilPreferredPeersHolder))
}

func TestNewListsSharder_NormalShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.P2pConfig.Sharding.TargetPeerCount = 25
	arg.P2pConfig.Sharding.MaxIntraShardValidators = 6
	arg.P2pConfig.Sharding.MaxCrossShardValidators = 5
	arg.P2pConfig.Sharding.MaxIntraShardObservers = 4
	arg.P2pConfig.Sharding.MaxCrossShardObservers = 3
	arg.P2pConfig.Sharding.MaxSeeders = 2
	arg.P2pConfig.Sharding.AdditionalConnections.MaxFullHistoryObservers = 1
	ls, err := networksharding.NewListsSharder(arg)

	assert.False(t, check.IfNil(ls))
	assert.Nil(t, err)
	assert.Equal(t, 25, ls.GetMaxPeerCount())
	assert.Equal(t, 6, ls.GetMaxIntraShardValidators())
	assert.Equal(t, 5, ls.GetMaxCrossShardValidators())
	assert.Equal(t, 4, ls.GetMaxIntraShardObservers())
	assert.Equal(t, 3, ls.GetMaxCrossShardObservers())
	assert.Equal(t, 2, ls.GetMaxSeeders())
	assert.Equal(t, 0, ls.GetMaxFullHistoryObservers())
	assert.Equal(t, 5, ls.GetMaxUnknown())
}

func TestNewListsSharder_FullArchiveShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.NodeOperationMode = p2p.FullArchiveMode
	arg.P2pConfig.Sharding.TargetPeerCount = 25
	arg.P2pConfig.Sharding.MaxIntraShardValidators = 6
	arg.P2pConfig.Sharding.MaxCrossShardValidators = 5
	arg.P2pConfig.Sharding.MaxIntraShardObservers = 4
	arg.P2pConfig.Sharding.MaxCrossShardObservers = 3
	arg.P2pConfig.Sharding.MaxSeeders = 2
	arg.P2pConfig.Sharding.AdditionalConnections.MaxFullHistoryObservers = 1
	ls, err := networksharding.NewListsSharder(arg)

	assert.False(t, check.IfNil(ls))
	assert.Nil(t, err)
	assert.Equal(t, 26, ls.GetMaxPeerCount())
	assert.Equal(t, 6, ls.GetMaxIntraShardValidators())
	assert.Equal(t, 5, ls.GetMaxCrossShardValidators())
	assert.Equal(t, 4, ls.GetMaxIntraShardObservers())
	assert.Equal(t, 3, ls.GetMaxCrossShardObservers())
	assert.Equal(t, 2, ls.GetMaxSeeders())
	assert.Equal(t, 1, ls.GetMaxFullHistoryObservers())
	assert.Equal(t, 5, ls.GetMaxUnknown())
}

// ------- ComputeEvictionList

func TestListsSharder_ComputeEvictionListNotReachedValidatorsShouldRetEmpty(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	ls, _ := networksharding.NewListsSharder(arg)
	pidCrtShard := peer.ID(fmt.Sprintf("%d %s", crtShardId, validatorMarker))
	pidCrossShard := peer.ID(fmt.Sprintf("%d %s", crossShardId, validatorMarker))
	pids := []peer.ID{pidCrtShard, pidCrossShard}

	evictList := ls.ComputeEvictionList(pids)

	assert.Equal(t, 0, len(evictList))
}

func TestListsSharder_ComputeEvictionListNotReachedObserversShouldRetEmpty(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	ls, _ := networksharding.NewListsSharder(arg)
	pidCrtShard := peer.ID(fmt.Sprintf("%d %s", crtShardId, observerMarker))
	pidCrossShard := peer.ID(fmt.Sprintf("%d %s", crossShardId, observerMarker))
	pids := []peer.ID{pidCrtShard, pidCrossShard}

	evictList := ls.ComputeEvictionList(pids)

	assert.Equal(t, 0, len(evictList))
}

func TestListsSharder_ComputeEvictionListNotReachedUnknownShouldRetEmpty(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	ls, _ := networksharding.NewListsSharder(arg)
	pidUnknown := peer.ID(fmt.Sprintf("0 %s", unknownMarker))
	pids := []peer.ID{pidUnknown}

	evictList := ls.ComputeEvictionList(pids)

	assert.Equal(t, 0, len(evictList))
}

func TestListsSharder_ComputeEvictionListReachedIntraShardShouldSortAndEvict(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	ls, _ := networksharding.NewListsSharder(arg)
	pidCrtShard1 := peer.ID(fmt.Sprintf("%d - 1 - %s", crtShardId, validatorMarker))
	pidCrtShard2 := peer.ID(fmt.Sprintf("%d - 2 - %s", crtShardId, validatorMarker))
	pids := []peer.ID{pidCrtShard2, pidCrtShard1}

	evictList := ls.ComputeEvictionList(pids)

	assert.Equal(t, 1, len(evictList))
	assert.Equal(t, pidCrtShard1, evictList[0])
}

func TestListsSharder_ComputeEvictionListUnknownPeersShouldFillTheGap(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.P2pConfig.Sharding.TargetPeerCount = 5
	ls, _ := networksharding.NewListsSharder(arg)

	unknownPids := make([]peer.ID, arg.P2pConfig.Sharding.TargetPeerCount)
	for i := 0; i < int(arg.P2pConfig.Sharding.TargetPeerCount); i++ {
		unknownPids[i] = unknownMarker
	}
	newUnknownPid := peer.ID(unknownMarker)
	unknownPids = append(unknownPids, newUnknownPid)

	evictList := ls.ComputeEvictionList(unknownPids)

	assert.Equal(t, 1, len(evictList))
	assert.Equal(t, unknownPids[0], evictList[0])
}

func TestListsSharder_ComputeEvictionListCrossShouldFillTheGap(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.P2pConfig.Sharding.TargetPeerCount = 5
	arg.P2pConfig.Sharding.MaxIntraShardValidators = 1
	arg.P2pConfig.Sharding.MaxCrossShardValidators = 1
	arg.P2pConfig.Sharding.MaxIntraShardObservers = 1
	arg.P2pConfig.Sharding.MaxCrossShardObservers = 1
	ls, _ := networksharding.NewListsSharder(arg)

	pids := []peer.ID{
		peer.ID(fmt.Sprintf("%d %s", crossShardId, validatorMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, validatorMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, observerMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, observerMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, unknownMarker)),
	}

	evictList := ls.ComputeEvictionList(pids)

	assert.Equal(t, 0, len(evictList))
}

func TestListsSharder_ComputeEvictionListEvictFromAllShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	arg.P2pConfig.Sharding.TargetPeerCount = 6
	arg.P2pConfig.Sharding.MaxIntraShardValidators = 1
	arg.P2pConfig.Sharding.MaxCrossShardValidators = 1
	arg.P2pConfig.Sharding.MaxIntraShardObservers = 1
	arg.P2pConfig.Sharding.MaxCrossShardObservers = 1
	arg.P2pConfig.Sharding.MaxSeeders = 1
	ls, _ := networksharding.NewListsSharder(arg)
	seeder := peer.ID(fmt.Sprintf("%d %s", crossShardId, seederMarker))
	ls.SetSeeders([]string{
		"ip6/" + seeder.String(),
	})

	pids := []peer.ID{
		peer.ID(fmt.Sprintf("%d %s", crtShardId, validatorMarker)),
		peer.ID(fmt.Sprintf("%d %s", crtShardId, validatorMarker)),

		peer.ID(fmt.Sprintf("%d %s", crossShardId, validatorMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, validatorMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, validatorMarker)),

		peer.ID(fmt.Sprintf("%d %s", crtShardId, observerMarker)),
		peer.ID(fmt.Sprintf("%d %s", crtShardId, observerMarker)),
		peer.ID(fmt.Sprintf("%d %s", crtShardId, observerMarker)),
		peer.ID(fmt.Sprintf("%d %s", crtShardId, observerMarker)),

		peer.ID(fmt.Sprintf("%d %s", crossShardId, observerMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, observerMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, observerMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, observerMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, observerMarker)),

		peer.ID(fmt.Sprintf("%d %s", crossShardId, unknownMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, unknownMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, unknownMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, unknownMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, unknownMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, unknownMarker)),

		peer.ID(fmt.Sprintf("%d %s", crossShardId, seederMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, seederMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, seederMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, seederMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, seederMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, seederMarker)),
		peer.ID(fmt.Sprintf("%d %s", crossShardId, seederMarker)),
	}

	evictList := ls.ComputeEvictionList(pids)

	assert.Equal(t, 21, len(evictList))
	assert.Equal(t, 1, countPeers(evictList, crtShardId, validatorMarker))
	assert.Equal(t, 2, countPeers(evictList, crossShardId, validatorMarker))
	assert.Equal(t, 3, countPeers(evictList, crtShardId, observerMarker))
	assert.Equal(t, 4, countPeers(evictList, crossShardId, observerMarker))
	assert.Equal(t, 5, countPeers(evictList, crossShardId, unknownMarker))
	assert.Equal(t, 6, countPeers(evictList, crossShardId, seederMarker))
}

func TestListsSharder_ComputeEvictionListShouldNotContainPreferredPeers(t *testing.T) {
	arg := createMockListSharderArguments()
	pids := []peer.ID{
		"preferredPeer0",
		"peer0",
		"peer1",
		"preferredPeer1",
		"peer2",
		"preferredPeer2",
	}
	arg.PreferredPeersHolder = &mock.PeersHolderStub{
		ContainsCalled: func(peerID core.PeerID) bool {
			return strings.HasPrefix(string(peerID), "preferred")
		},
	}

	ls, _ := networksharding.NewListsSharder(arg)
	seeder := peer.ID(fmt.Sprintf("%d %s", crossShardId, seederMarker))
	ls.SetSeeders([]string{
		"ip6/" + seeder.String(),
	})

	evictList := ls.ComputeEvictionList(pids)

	for _, peerID := range evictList {
		require.False(t, strings.HasPrefix(string(peerID), "preferred"))
	}
}

func TestListsSharder_ComputeEvictionListWithRealPreferredPeersHandler(t *testing.T) {
	arg := createMockListSharderArguments()

	preferredHexPrefix := "preferred"
	prefP0 := preferredHexPrefix + "preferredPeer0"
	prefP1 := preferredHexPrefix + "preferredPeer1"
	prefP2 := preferredHexPrefix + "preferredPeer2"
	pubKeyHexSuffix := hex.EncodeToString([]byte("pubKey"))
	pids := []peer.ID{
		peer.ID(core.PeerID(prefP0).Pretty()),
		"peer0",
		"peer1",
		peer.ID(core.PeerID(prefP1).Pretty()),
		"peer2",
		peer.ID(core.PeerID(prefP2).Pretty()),
	}

	prefPeers := []string{
		core.PeerID(prefP0).Pretty(),
		core.PeerID(prefP1).Pretty(),
		core.PeerID(prefP2).Pretty(),
	}

	arg.PreferredPeersHolder, _ = peersHolder.NewPeersHolder(prefPeers)
	for _, prefPid := range prefPeers {
		peerId := core.PeerID(prefPid)
		arg.PreferredPeersHolder.PutConnectionAddress(peerId, prefPid)
		arg.PreferredPeersHolder.PutShardID(peerId, 0)
	}

	arg.PeerResolver = &mock.PeerShardResolverStub{
		GetPeerInfoCalled: func(pid core.PeerID) core.P2PPeerInfo {
			if strings.HasPrefix(string(pid), preferredHexPrefix) {
				pkBytes, _ := hex.DecodeString(string(pid) + pubKeyHexSuffix)
				return core.P2PPeerInfo{
					PeerType:    0,
					PeerSubType: 0,
					ShardID:     0,
					PkBytes:     pkBytes,
				}
			}
			return core.P2PPeerInfo{}
		},
	}
	ls, _ := networksharding.NewListsSharder(arg)
	seeder := peer.ID(fmt.Sprintf("%d %s", crossShardId, seederMarker))
	ls.SetSeeders([]string{
		"ip6/" + seeder.String(),
	})

	evictList := ls.ComputeEvictionList(pids)
	for _, peerID := range evictList {
		require.False(t, strings.HasPrefix(string(peerID), preferredHexPrefix))
	}

	found := arg.PreferredPeersHolder.Contains(core.PeerID(peer.ID(prefP0).String()))
	require.True(t, found)

	found = arg.PreferredPeersHolder.Contains(core.PeerID(peer.ID(prefP1).String()))
	require.True(t, found)

	found = arg.PreferredPeersHolder.Contains(core.PeerID(peer.ID(prefP2).String()))
	require.True(t, found)

	peers := arg.PreferredPeersHolder.Get()
	expectedMap := map[uint32][]core.PeerID{
		0: {
			core.PeerID(peer.ID(prefP0).String()),
			core.PeerID(peer.ID(prefP1).String()),
			core.PeerID(peer.ID(prefP2).String()),
		},
	}
	require.Equal(t, expectedMap, peers)
}

// ------- Has

func TestListsSharder_HasNotFound(t *testing.T) {
	t.Parallel()

	list := []peer.ID{"pid1", "pid2", "pid3"}
	arg := createMockListSharderArguments()
	ls, _ := networksharding.NewListsSharder(arg)

	assert.False(t, ls.Has("pid4", list))
}

func TestListsSharder_HasEmpty(t *testing.T) {
	t.Parallel()

	list := make([]peer.ID, 0)
	arg := createMockListSharderArguments()
	ls, _ := networksharding.NewListsSharder(arg)

	assert.False(t, ls.Has("pid4", list))
}

func TestListsSharder_HasFound(t *testing.T) {
	t.Parallel()

	list := []peer.ID{"pid1", "pid2", "pid3"}
	arg := createMockListSharderArguments()
	ls, _ := networksharding.NewListsSharder(arg)

	assert.True(t, ls.Has("pid2", list))
}

// ------- computeDistance

func TestComputeDistanceByCountingBits(t *testing.T) {
	t.Parallel()

	// compute will be done on hashes. Impossible to predict the outcome in this test
	assert.Equal(t, uint64(0), networksharding.ComputeDistanceByCountingBits("", "").Uint64())
	assert.Equal(t, uint64(0), networksharding.ComputeDistanceByCountingBits("a", "a").Uint64())
	assert.Equal(t, uint64(139), networksharding.ComputeDistanceByCountingBits(peer.ID([]byte{0}), peer.ID([]byte{1})).Uint64())
	assert.Equal(t, uint64(130), networksharding.ComputeDistanceByCountingBits(peer.ID([]byte{0}), peer.ID([]byte{255})).Uint64())
	assert.Equal(t, uint64(117), networksharding.ComputeDistanceByCountingBits(peer.ID([]byte{0, 128}), peer.ID([]byte{255, 255})).Uint64())
}

func TestComputeDistanceLog2Based(t *testing.T) {
	t.Parallel()

	// compute will be done on hashes. Impossible to predict the outcome in this test
	assert.Equal(t, uint64(0), networksharding.ComputeDistanceLog2Based("", "").Uint64())
	assert.Equal(t, uint64(0), networksharding.ComputeDistanceLog2Based("a", "a").Uint64())
	assert.Equal(t, uint64(254), networksharding.ComputeDistanceLog2Based(peer.ID([]byte{0}), peer.ID([]byte{1})).Uint64())
	assert.Equal(t, uint64(250), networksharding.ComputeDistanceLog2Based(peer.ID([]byte{254}), peer.ID([]byte{255})).Uint64())
	assert.Equal(t, uint64(256), networksharding.ComputeDistanceLog2Based(peer.ID([]byte{0, 128}), peer.ID([]byte{255, 255})).Uint64())
}

func TestListsSharder_SetPeerShardResolverNilShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	ls, _ := networksharding.NewListsSharder(arg)

	err := ls.SetPeerShardResolver(nil)

	assert.Equal(t, p2p.ErrNilPeerShardResolver, err)
}

func TestListsSharder_SetPeerShardResolverShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockListSharderArguments()
	ls, _ := networksharding.NewListsSharder(arg)
	newPeerShardResolver := &mock.PeerShardResolverStub{}
	err := ls.SetPeerShardResolver(newPeerShardResolver)

	// pointer testing
	assert.True(t, ls.GetPeerShardResolver() == newPeerShardResolver)
	assert.Nil(t, err)
}
