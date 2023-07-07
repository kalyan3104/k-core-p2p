package factory_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kalyan3104/k-core-p2p/config"
	"github.com/kalyan3104/k-core-p2p/libp2p/networksharding"
	"github.com/kalyan3104/k-core-p2p/libp2p/networksharding/factory"
	"github.com/kalyan3104/k-core-p2p/mock"
	"github.com/kalyan3104/k-core/core/check"
	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/stretchr/testify/assert"
)

func createMockArg() factory.ArgsSharderFactory {
	return factory.ArgsSharderFactory{

		PeerShardResolver:    &mock.PeerShardResolverStub{},
		Pid:                  "",
		PreferredPeersHolder: &mock.PeersHolderStub{},
		P2pConfig: config.P2PConfig{
			Sharding: config.ShardingConfig{
				Type:                    "unknown",
				TargetPeerCount:         6,
				MaxIntraShardValidators: 1,
				MaxCrossShardValidators: 1,
				MaxIntraShardObservers:  1,
				MaxCrossShardObservers:  1,
				AdditionalConnections: config.AdditionalConnectionsConfig{
					MaxFullHistoryObservers: 1,
				},
			},
		},
		NodeOperationMode: p2p.NormalOperation,
	}
}

func TestNewSharder_CreateListsSharderUnknownNodeOperationShouldError(t *testing.T) {
	t.Parallel()

	arg := createMockArg()
	arg.P2pConfig.Sharding.Type = p2p.ListsSharder
	arg.NodeOperationMode = ""
	sharder, err := factory.NewSharder(arg)

	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
	assert.True(t, strings.Contains(err.Error(), "unknown node operation mode"))
	assert.True(t, check.IfNil(sharder))
}

func TestNewSharder_CreateListsSharderShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockArg()
	arg.P2pConfig.Sharding.Type = p2p.ListsSharder
	sharder, err := factory.NewSharder(arg)
	maxPeerCount := uint32(5)
	maxValidators := uint32(1)
	maxObservers := uint32(1)

	argListsSharder := networksharding.ArgListsSharder{
		PeerResolver: &mock.PeerShardResolverStub{},
		SelfPeerId:   "",
		P2pConfig: config.P2PConfig{
			Sharding: config.ShardingConfig{
				TargetPeerCount:         maxPeerCount,
				MaxIntraShardObservers:  maxObservers,
				MaxIntraShardValidators: maxValidators,
				MaxCrossShardObservers:  maxObservers,
				MaxCrossShardValidators: maxValidators,
				MaxSeeders:              0,
			},
		},
	}
	expectedSharder, _ := networksharding.NewListsSharder(argListsSharder)
	assert.Nil(t, err)
	assert.IsType(t, reflect.TypeOf(expectedSharder), reflect.TypeOf(sharder))
}

func TestNewSharder_CreateOneListSharderShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockArg()
	arg.P2pConfig.Sharding.Type = p2p.OneListSharder
	sharder, err := factory.NewSharder(arg)
	maxPeerCount := 2

	expectedSharder, _ := networksharding.NewOneListSharder("", maxPeerCount)
	assert.Nil(t, err)
	assert.IsType(t, reflect.TypeOf(expectedSharder), reflect.TypeOf(sharder))
}

func TestNewSharder_CreateNilListSharderShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockArg()
	arg.P2pConfig.Sharding.Type = p2p.NilListSharder
	sharder, err := factory.NewSharder(arg)

	expectedSharder := networksharding.NewNilListSharder()
	assert.Nil(t, err)
	assert.IsType(t, reflect.TypeOf(expectedSharder), reflect.TypeOf(sharder))
}

func TestNewSharder_CreateWithUnknownVariantShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArg()
	sharder, err := factory.NewSharder(arg)

	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
	assert.True(t, check.IfNil(sharder))
}
