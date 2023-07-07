package mock

import (
	"github.com/kalyan3104/k-core-p2p/libp2p"
)

// ChannelLoadBalancerStub -
type ChannelLoadBalancerStub struct {
	AddChannelCalled                    func(pipe string) error
	RemoveChannelCalled                 func(pipe string) error
	GetChannelOrDefaultCalled           func(pipe string) chan *libp2p.SendableData
	CollectOneElementFromChannelsCalled func() *libp2p.SendableData
	CloseCalled                         func() error
}

// AddChannel -
func (clbs *ChannelLoadBalancerStub) AddChannel(pipe string) error {
	return clbs.AddChannelCalled(pipe)
}

// RemoveChannel -
func (clbs *ChannelLoadBalancerStub) RemoveChannel(pipe string) error {
	return clbs.RemoveChannelCalled(pipe)
}

// GetChannelOrDefault -
func (clbs *ChannelLoadBalancerStub) GetChannelOrDefault(pipe string) chan *libp2p.SendableData {
	return clbs.GetChannelOrDefaultCalled(pipe)
}

// CollectOneElementFromChannels -
func (clbs *ChannelLoadBalancerStub) CollectOneElementFromChannels() *libp2p.SendableData {
	return clbs.CollectOneElementFromChannelsCalled()
}

// Close -
func (clbs *ChannelLoadBalancerStub) Close() error {
	if clbs.CloseCalled != nil {
		return clbs.CloseCalled()
	}

	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (clbs *ChannelLoadBalancerStub) IsInterfaceNil() bool {
	return clbs == nil
}
