package libp2p_test

import (
	"errors"
	"testing"

	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/kalyan3104/k-core-p2p/libp2p"
	"github.com/kalyan3104/k-core-p2p/mock"
	"github.com/kalyan3104/k-core/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTopicProcessors(t *testing.T) {
	t.Parallel()

	tp := libp2p.NewTopicProcessors()

	assert.NotNil(t, tp)
}

func TestTopicProcessorsAddShouldWork(t *testing.T) {
	t.Parallel()

	tp := libp2p.NewTopicProcessors()

	identifier := "identifier"
	proc := &mock.MessageProcessorStub{}
	err := tp.AddTopicProcessor(identifier, proc)

	assert.Nil(t, err)
	topics, processors := tp.GetList()
	require.Equal(t, 1, len(topics))
	require.Equal(t, 1, len(processors))
	assert.True(t, proc == processors[0]) // pointer testing
}

func TestTopicProcessorsDoubleAddShouldErr(t *testing.T) {
	t.Parallel()

	tp := libp2p.NewTopicProcessors()

	identifier := "identifier"
	_ = tp.AddTopicProcessor(identifier, &mock.MessageProcessorStub{})
	err := tp.AddTopicProcessor(identifier, &mock.MessageProcessorStub{})

	assert.True(t, errors.Is(err, p2p.ErrMessageProcessorAlreadyDefined))
	topics, processors := tp.GetList()
	require.Equal(t, 1, len(topics))
	require.Equal(t, 1, len(processors))
}

func TestTopicProcessorsRemoveInexistentShouldErr(t *testing.T) {
	t.Parallel()

	tp := libp2p.NewTopicProcessors()

	identifier := "identifier"
	err := tp.RemoveTopicProcessor(identifier)

	assert.True(t, errors.Is(err, p2p.ErrMessageProcessorDoesNotExists))
}

func TestTopicProcessorsRemoveShouldWork(t *testing.T) {
	t.Parallel()

	tp := libp2p.NewTopicProcessors()

	identifier1 := "identifier1"
	identifier2 := "identifier2"
	_ = tp.AddTopicProcessor(identifier1, &mock.MessageProcessorStub{})
	_ = tp.AddTopicProcessor(identifier2, &mock.MessageProcessorStub{})

	topics, processors := tp.GetList()
	require.Equal(t, 2, len(topics))
	require.Equal(t, 2, len(processors))

	err := tp.RemoveTopicProcessor(identifier2)
	assert.Nil(t, err)

	topics, processors = tp.GetList()
	require.Equal(t, 1, len(topics))
	require.Equal(t, 1, len(processors))

	err = tp.RemoveTopicProcessor(identifier1)
	assert.Nil(t, err)

	topics, processors = tp.GetList()
	require.Equal(t, 0, len(topics))
	require.Equal(t, 0, len(processors))
}

func TestTopicProcessorsGetListShouldWorkAndPreserveOrder(t *testing.T) {
	t.Parallel()

	tp := libp2p.NewTopicProcessors()

	identifier1 := "identifier1"
	identifier2 := "identifier2"
	identifier3 := "identifier3"
	handler1 := &mock.MessageProcessorStub{
		ProcessMessageCalled: func(message p2p.MessageP2P, fromConnectedPeer core.PeerID) error {
			return nil
		},
	}
	handler2 := &mock.MessageProcessorStub{
		ProcessMessageCalled: func(message p2p.MessageP2P, fromConnectedPeer core.PeerID) error {
			return nil
		},
	}
	handler3 := &mock.MessageProcessorStub{
		ProcessMessageCalled: func(message p2p.MessageP2P, fromConnectedPeer core.PeerID) error {
			return nil
		},
	}

	_ = tp.AddTopicProcessor(identifier3, handler3)
	_ = tp.AddTopicProcessor(identifier1, handler1)
	_ = tp.AddTopicProcessor(identifier2, handler2)

	topics, processors := tp.GetList()
	assert.ElementsMatch(t, []string{identifier1, identifier2, identifier3}, topics)
	assert.ElementsMatch(t, []p2p.MessageProcessor{handler1, handler2, handler3}, processors)

	_ = tp.RemoveTopicProcessor(identifier1)
	topics, processors = tp.GetList()
	assert.ElementsMatch(t, []string{identifier2, identifier3}, topics)
	assert.ElementsMatch(t, []p2p.MessageProcessor{handler2, handler3}, processors)

	_ = tp.RemoveTopicProcessor(identifier2)
	topics, processors = tp.GetList()
	assert.Equal(t, []string{identifier3}, topics)
	assert.Equal(t, []p2p.MessageProcessor{handler3}, processors)

	_ = tp.RemoveTopicProcessor(identifier3)
	topics, processors = tp.GetList()
	assert.Empty(t, topics)
	assert.Empty(t, processors)
}
