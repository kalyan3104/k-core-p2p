package networksharding_test

import (
	"errors"
	"testing"

	"github.com/kalyan3104/k-core-p2p/libp2p/networksharding"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/libp2p/go-libp2p/core/peer"
	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/stretchr/testify/assert"
)

func TestNewOneListSharder_InvalidMaxPeerCountShouldErr(t *testing.T) {
	t.Parallel()

	ols, err := networksharding.NewOneListSharder(
		"",
		networksharding.MinAllowedConnectedPeersOneSharder-1,
	)

	assert.True(t, check.IfNil(ols))
	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
}

func TestNewOneListSharder_ShouldWork(t *testing.T) {
	t.Parallel()

	ols, err := networksharding.NewOneListSharder(
		"",
		networksharding.MinAllowedConnectedPeersOneSharder,
	)

	assert.False(t, check.IfNil(ols))
	assert.Nil(t, err)
}

// ------- ComputeEvictionList

func TestOneListSharder_ComputeEvictionListNotReachedShouldRetEmpty(t *testing.T) {
	t.Parallel()

	ols, _ := networksharding.NewOneListSharder(
		crtPid,
		networksharding.MinAllowedConnectedPeersOneSharder,
	)
	pid1 := peer.ID("pid1")
	pid2 := peer.ID("pid2")
	pids := []peer.ID{pid1, pid2}

	evictList := ols.ComputeEvictionList(pids)

	assert.Equal(t, 0, len(evictList))
}

func TestOneListSharder_ComputeEvictionListReachedIntraShardShouldSortAndEvict(t *testing.T) {
	t.Parallel()

	ols, _ := networksharding.NewOneListSharder(
		crtPid,
		networksharding.MinAllowedConnectedPeersOneSharder,
	)
	pid1 := peer.ID("pid1")
	pid2 := peer.ID("pid2")
	pid3 := peer.ID("pid3")
	pid4 := peer.ID("pid4")
	pids := []peer.ID{pid1, pid2, pid3, pid4}

	evictList := ols.ComputeEvictionList(pids)

	assert.Equal(t, 1, len(evictList))
	assert.Equal(t, pid3, evictList[0])
}

// ------- Has

func TestOneListSharder_HasNotFound(t *testing.T) {
	t.Parallel()

	list := []peer.ID{"pid1", "pid2", "pid3"}
	ols, _ := networksharding.NewOneListSharder(
		crtPid,
		networksharding.MinAllowedConnectedPeersOneSharder,
	)

	assert.False(t, ols.Has("pid4", list))
}

func TestOneListSharder_HasEmpty(t *testing.T) {
	t.Parallel()

	list := make([]peer.ID, 0)
	ols, _ := networksharding.NewOneListSharder(
		crtPid,
		networksharding.MinAllowedConnectedPeersOneSharder,
	)

	assert.False(t, ols.Has("pid4", list))
}

func TestOneListSharder_HasFound(t *testing.T) {
	t.Parallel()

	list := []peer.ID{"pid1", "pid2", "pid3"}
	ols, _ := networksharding.NewOneListSharder(
		crtPid,
		networksharding.MinAllowedConnectedPeersOneSharder,
	)

	assert.True(t, ols.Has("pid2", list))
}

func TestOneListSharder_SetPeerShardResolverShouldNotPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r != nil {
			assert.Fail(t, "should not have paniced")
		}
	}()

	ols, _ := networksharding.NewOneListSharder(
		"",
		networksharding.MinAllowedConnectedPeersOneSharder,
	)

	err := ols.SetPeerShardResolver(nil)

	assert.Nil(t, err)
}
