package libp2p_test

import (
	"bytes"
	"testing"

	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/kalyan3104/k-core-p2p/libp2p"
	"github.com/kalyan3104/k-core-p2p/mock"
	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func createStubConn() *mock.ConnStub {
	return &mock.ConnStub{
		RemotePeerCalled: func() peer.ID {
			return "remote peer"
		},
	}
}

func TestNewConnectionMonitorWrapper_ShouldWork(t *testing.T) {
	t.Parallel()

	cmw := libp2p.NewConnectionMonitorWrapper(
		&mock.NetworkStub{},
		&mock.ConnectionMonitorStub{},
		&mock.PeerDenialEvaluatorStub{},
	)

	assert.False(t, check.IfNil(cmw))
}

// ------- Connected

func TestConnectionMonitorNotifier_ConnectedBlackListedShouldCallClose(t *testing.T) {
	t.Parallel()

	peerCloseCalled := false
	conn := createStubConn()
	conn.CloseCalled = func() error {
		peerCloseCalled = true

		return nil
	}
	networkInstance := &mock.NetworkStub{}
	cmw := libp2p.NewConnectionMonitorWrapper(
		networkInstance,
		&mock.ConnectionMonitorStub{},
		&mock.PeerDenialEvaluatorStub{
			IsDeniedCalled: func(pid core.PeerID) bool {
				return true
			},
		},
	)

	cmw.Connected(networkInstance, conn)

	assert.True(t, peerCloseCalled)
}

func TestConnectionMonitorNotifier_ConnectedNotBlackListedShouldCallConnected(t *testing.T) {
	t.Parallel()

	peerConnectedCalled := false
	conn := createStubConn()
	networkInstance := &mock.NetworkStub{}
	cmw := libp2p.NewConnectionMonitorWrapper(
		networkInstance,
		&mock.ConnectionMonitorStub{
			ConnectedCalled: func(netw network.Network, conn network.Conn) {
				peerConnectedCalled = true
			},
		},
		&mock.PeerDenialEvaluatorStub{
			IsDeniedCalled: func(pid core.PeerID) bool {
				return false
			},
		},
	)

	cmw.Connected(networkInstance, conn)

	assert.True(t, peerConnectedCalled)
}

// ------- Functions

func TestConnectionMonitorNotifier_FunctionsShouldCallHandler(t *testing.T) {
	t.Parallel()

	listenCalled := false
	listenCloseCalled := false
	disconnectCalled := false
	cmw := libp2p.NewConnectionMonitorWrapper(
		&mock.NetworkStub{},
		&mock.ConnectionMonitorStub{
			ListenCalled: func(network.Network, multiaddr.Multiaddr) {
				listenCalled = true
			},
			ListenCloseCalled: func(network.Network, multiaddr.Multiaddr) {
				listenCloseCalled = true
			},
			DisconnectedCalled: func(network.Network, network.Conn) {
				disconnectCalled = true
			},
		},
		&mock.PeerDenialEvaluatorStub{},
	)

	cmw.Listen(nil, nil)
	cmw.ListenClose(nil, nil)
	cmw.Disconnected(nil, nil)

	assert.True(t, listenCalled)
	assert.True(t, listenCloseCalled)
	assert.True(t, disconnectCalled)
}

// ------- SetBlackListHandler

func TestConnectionMonitorWrapper_SetBlackListHandlerNilHandlerShouldErr(t *testing.T) {
	t.Parallel()

	cmw := libp2p.NewConnectionMonitorWrapper(
		&mock.NetworkStub{},
		&mock.ConnectionMonitorStub{},
		&mock.PeerDenialEvaluatorStub{},
	)

	err := cmw.SetPeerDenialEvaluator(nil)

	assert.Equal(t, p2p.ErrNilPeerDenialEvaluator, err)
}

func TestConnectionMonitorWrapper_SetBlackListHandlerShouldWork(t *testing.T) {
	t.Parallel()

	cmw := libp2p.NewConnectionMonitorWrapper(
		&mock.NetworkStub{},
		&mock.ConnectionMonitorStub{},
		&mock.PeerDenialEvaluatorStub{},
	)
	newPeerDenialEvaluator := &mock.PeerDenialEvaluatorStub{}

	err := cmw.SetPeerDenialEvaluator(newPeerDenialEvaluator)

	assert.Nil(t, err)
	// pointer testing
	assert.True(t, newPeerDenialEvaluator == cmw.PeerDenialEvaluator())
}

// ------- CheckConnectionsBlocking

func TestConnectionMonitorWrapper_CheckConnectionsBlockingShouldWork(t *testing.T) {
	t.Parallel()

	whiteListPeer := peer.ID("whitelisted")
	blackListPeer := peer.ID("blacklisted")
	closeCalled := 0
	cmw := libp2p.NewConnectionMonitorWrapper(
		&mock.NetworkStub{
			PeersCall: func() []peer.ID {
				return []peer.ID{whiteListPeer, blackListPeer}
			},
			ClosePeerCall: func(id peer.ID) error {
				if id == blackListPeer {
					closeCalled++
					return nil
				}
				assert.Fail(t, "should have called only the black listed peer ")

				return nil
			},
		},
		&mock.ConnectionMonitorStub{},
		&mock.PeerDenialEvaluatorStub{
			IsDeniedCalled: func(pid core.PeerID) bool {
				return bytes.Equal(core.PeerID(blackListPeer).Bytes(), pid.Bytes())
			},
		},
	)

	cmw.CheckConnectionsBlocking()
	assert.Equal(t, 1, closeCalled)
}
