package message_test

import (
	"testing"

	"github.com/kalyan3104/k-core-p2p/message"
	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/stretchr/testify/assert"
)

func TestMessage_AllFieldsShouldWork(t *testing.T) {
	t.Parallel()

	from := []byte("from")
	data := []byte("data")
	seqNo := []byte("seq no")
	topic := "topic"
	sig := []byte("sig")
	key := []byte("key")
	peer := core.PeerID("peer")

	msg := &message.Message{
		FromField:      from,
		DataField:      data,
		SeqNoField:     seqNo,
		TopicField:     topic,
		SignatureField: sig,
		KeyField:       key,
		PeerField:      peer,
	}

	assert.False(t, check.IfNil(msg))
	assert.Equal(t, from, msg.From())
	assert.Equal(t, data, msg.Data())
	assert.Equal(t, seqNo, msg.SeqNo())
	assert.Equal(t, topic, msg.Topic())
	assert.Equal(t, sig, msg.Signature())
	assert.Equal(t, key, msg.Key())
	assert.Equal(t, peer, msg.Peer())
}
