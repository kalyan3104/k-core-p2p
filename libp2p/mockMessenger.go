package libp2p

import (
	"context"

	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/kalyan3104/k-core-p2p/libp2p/crypto"
	"github.com/kalyan3104/k-core-p2p/libp2p/metrics/factory"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

// NewMockMessenger creates a new sandbox testable instance of libP2P messenger
// It should not open ports on current machine
// Should be used only in testing!
func NewMockMessenger(
	args ArgsNetworkMessenger,
	mockNet mocknet.Mocknet,
) (*networkMessenger, error) {
	if mockNet == nil {
		return nil, p2p.ErrNilMockNet
	}

	h, err := mockNet.GenPeer()
	if err != nil {
		return nil, err
	}

	p2pSignerArgs := crypto.ArgsP2pSignerWrapper{
		PrivateKey: args.P2pPrivateKey,
		Signer:     args.P2pSingleSigner,
		KeyGen:     args.P2pKeyGenerator,
	}

	signer, err := crypto.NewP2PSignerWrapper(p2pSignerArgs)
	if err != nil {
		return nil, err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	p2pNode := &networkMessenger{
		p2pSigner:  signer,
		p2pHost:    NewConnectableHost(h),
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}
	p2pNode.printConnectionsWatcher, err = factory.NewConnectionsWatcher(args.ConnectionWatcherType, ttlConnectionsWatcher)
	if err != nil {
		return nil, err
	}

	err = addComponentsToNode(args, p2pNode, withoutMessageSigning)
	if err != nil {
		return nil, err
	}

	return p2pNode, err
}
