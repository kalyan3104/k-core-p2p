package libp2p_test

import (
	"sync/atomic"
	"testing"
	"time"

	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/kalyan3104/k-core-p2p/libp2p"
	"github.com/kalyan3104/k-core-p2p/mock"
	"github.com/kalyan3104/k-core/core"
	coreAtomic "github.com/kalyan3104/k-core/core/atomic"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func TestNewPeersOnChannel_NilPeersRatingHandlerShouldErr(t *testing.T) {
	t.Parallel()

	poc, err := libp2p.NewPeersOnChannel(nil, nil, 1, 1)

	assert.Nil(t, poc)
	assert.Equal(t, p2p.ErrNilPeersRatingHandler, err)
}

func TestNewPeersOnChannel_NilFetchPeersHandlerShouldErr(t *testing.T) {
	t.Parallel()

	poc, err := libp2p.NewPeersOnChannel(&mock.PeersRatingHandlerStub{}, nil, 1, 1)

	assert.Nil(t, poc)
	assert.Equal(t, p2p.ErrNilFetchPeersOnTopicHandler, err)
}

func TestNewPeersOnChannel_InvalidRefreshIntervalShouldErr(t *testing.T) {
	t.Parallel()

	poc, err := libp2p.NewPeersOnChannel(
		&mock.PeersRatingHandlerStub{},
		func(topic string) []peer.ID {
			return nil
		},
		0,
		1)

	assert.Nil(t, poc)
	assert.Equal(t, p2p.ErrInvalidDurationProvided, err)
}

func TestNewPeersOnChannel_InvalidTTLIntervalShouldErr(t *testing.T) {
	t.Parallel()

	poc, err := libp2p.NewPeersOnChannel(
		&mock.PeersRatingHandlerStub{},
		func(topic string) []peer.ID {
			return nil
		},
		1,
		0)

	assert.Nil(t, poc)
	assert.Equal(t, p2p.ErrInvalidDurationProvided, err)
}

func TestNewPeersOnChannel_OkValsShouldWork(t *testing.T) {
	t.Parallel()

	poc, err := libp2p.NewPeersOnChannel(
		&mock.PeersRatingHandlerStub{},
		func(topic string) []peer.ID {
			return nil
		},
		1,
		1)

	assert.NotNil(t, poc)
	assert.Nil(t, err)
}

func TestPeersOnChannel_ConnectedPeersOnChannelMissingTopicShouldTriggerFetchAndReturn(t *testing.T) {
	t.Parallel()

	retPeerIDs := []peer.ID{"peer1", "peer2"}
	wasFetchCalled := atomic.Value{}
	wasFetchCalled.Store(false)

	poc, _ := libp2p.NewPeersOnChannel(
		&mock.PeersRatingHandlerStub{},
		func(topic string) []peer.ID {
			if topic == testTopic {
				wasFetchCalled.Store(true)
				return retPeerIDs
			}
			return nil
		},
		time.Second,
		time.Second,
	)

	peers := poc.ConnectedPeersOnChannel(testTopic)

	assert.True(t, wasFetchCalled.Load().(bool))
	for idx, pid := range retPeerIDs {
		assert.Equal(t, []byte(pid), peers[idx].Bytes())
	}
}

func TestPeersOnChannel_ConnectedPeersOnChannelFindTopicShouldReturn(t *testing.T) {
	t.Parallel()

	retPeerIDs := []core.PeerID{"peer1", "peer2"}
	wasFetchCalled := atomic.Value{}
	wasFetchCalled.Store(false)

	poc, _ := libp2p.NewPeersOnChannel(
		&mock.PeersRatingHandlerStub{},
		func(topic string) []peer.ID {
			wasFetchCalled.Store(true)
			return nil
		},
		time.Second,
		time.Second,
	)
	// manually put peers
	poc.SetPeersOnTopic(testTopic, time.Now(), retPeerIDs)

	peers := poc.ConnectedPeersOnChannel(testTopic)

	assert.False(t, wasFetchCalled.Load().(bool))
	for idx, pid := range retPeerIDs {
		assert.Equal(t, []byte(pid), peers[idx].Bytes())
	}
}

func TestPeersOnChannel_RefreshShouldBeDone(t *testing.T) {
	t.Parallel()

	retPeerIDs := []core.PeerID{"peer1", "peer2"}
	wasFetchCalled := coreAtomic.Flag{}
	wasFetchCalled.Reset()

	refreshInterval := time.Millisecond * 100
	ttlInterval := time.Duration(2)

	poc, _ := libp2p.NewPeersOnChannel(
		&mock.PeersRatingHandlerStub{},
		func(topic string) []peer.ID {
			wasFetchCalled.SetValue(true)
			return nil
		},
		refreshInterval,
		ttlInterval,
	)
	poc.SetTimeHandler(func() time.Time {
		return time.Unix(0, 4)
	})
	// manually put peers
	poc.SetPeersOnTopic(testTopic, time.Unix(0, 1), retPeerIDs)

	// wait for the go routine cycle finish up
	time.Sleep(time.Second)

	assert.True(t, wasFetchCalled.IsSet())
	assert.Empty(t, poc.GetPeers(testTopic))
}
