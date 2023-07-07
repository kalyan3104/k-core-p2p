package libp2p_test

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"

	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/kalyan3104/k-core-p2p/data"
	"github.com/kalyan3104/k-core-p2p/libp2p"
	"github.com/kalyan3104/k-core-p2p/mock"
	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/libp2p/go-libp2p-pubsub"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	libp2pCrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getRandomID() []byte {
	prvKey, _, _ := libp2pCrypto.GenerateSecp256k1Key(rand.Reader)
	id, _ := peer.IDFromPublicKey(prvKey.GetPublic())

	return []byte(id)
}

func TestMessage_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	pMes := &pubsub.Message{}
	m, err := libp2p.NewMessage(pMes, nil)

	assert.True(t, check.IfNil(m))
	assert.True(t, errors.Is(err, p2p.ErrNilMarshalizer))
}

func TestMessage_ShouldErrBecauseOfFromField(t *testing.T) {
	t.Parallel()

	from := []byte("dummy from")
	marshalizer := &mock.ProtoMarshallerMock{}

	topicMessage := &data.TopicMessage{
		Version:   libp2p.CurrentTopicMessageVersion,
		Timestamp: time.Now().Unix(),
		Payload:   []byte("data"),
	}
	buff, _ := marshalizer.Marshal(topicMessage)
	topic := "topic"
	mes := &pb.Message{
		From:  from,
		Data:  buff,
		Topic: &topic,
	}
	pMes := &pubsub.Message{Message: mes}
	m, err := libp2p.NewMessage(pMes, marshalizer)

	assert.True(t, check.IfNil(m))
	assert.NotNil(t, err)
}

func TestMessage_ShouldWork(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.ProtoMarshallerMock{}
	topicMessage := &data.TopicMessage{
		Version:   libp2p.CurrentTopicMessageVersion,
		Timestamp: time.Now().Unix(),
		Payload:   []byte("data"),
	}
	buff, _ := marshalizer.Marshal(topicMessage)
	topic := "topic"
	mes := &pb.Message{
		From:  getRandomID(),
		Data:  buff,
		Topic: &topic,
	}

	pMes := &pubsub.Message{Message: mes}
	m, err := libp2p.NewMessage(pMes, marshalizer)

	require.Nil(t, err)
	assert.False(t, check.IfNil(m))
}

func TestMessage_From(t *testing.T) {
	t.Parallel()

	from := getRandomID()
	marshalizer := &mock.ProtoMarshallerMock{}
	topicMessage := &data.TopicMessage{
		Version:   libp2p.CurrentTopicMessageVersion,
		Timestamp: time.Now().Unix(),
		Payload:   []byte("data"),
	}
	buff, _ := marshalizer.Marshal(topicMessage)
	topic := "topic"
	mes := &pb.Message{
		From:  from,
		Data:  buff,
		Topic: &topic,
	}
	pMes := &pubsub.Message{Message: mes}
	m, err := libp2p.NewMessage(pMes, marshalizer)

	require.Nil(t, err)
	assert.Equal(t, m.From(), from)
}

func TestMessage_Peer(t *testing.T) {
	t.Parallel()

	id := getRandomID()
	marshalizer := &mock.ProtoMarshallerMock{}

	topicMessage := &data.TopicMessage{
		Version:   libp2p.CurrentTopicMessageVersion,
		Timestamp: time.Now().Unix(),
		Payload:   []byte("data"),
	}
	buff, _ := marshalizer.Marshal(topicMessage)
	topic := "topic"
	mes := &pb.Message{
		From:  id,
		Data:  buff,
		Topic: &topic,
	}
	pMes := &pubsub.Message{Message: mes}
	m, err := libp2p.NewMessage(pMes, marshalizer)

	require.Nil(t, err)
	assert.Equal(t, core.PeerID(id), m.Peer())
}

func TestMessage_WrongVersionShouldErr(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.ProtoMarshallerMock{}

	topicMessage := &data.TopicMessage{
		Version:   libp2p.CurrentTopicMessageVersion + 1,
		Timestamp: time.Now().Unix(),
		Payload:   []byte("data"),
	}
	buff, _ := marshalizer.Marshal(topicMessage)
	topic := "topic"
	mes := &pb.Message{
		From:  getRandomID(),
		Data:  buff,
		Topic: &topic,
	}

	pMes := &pubsub.Message{Message: mes}
	m, err := libp2p.NewMessage(pMes, marshalizer)

	assert.True(t, check.IfNil(m))
	assert.True(t, errors.Is(err, p2p.ErrUnsupportedMessageVersion))
}

func TestMessage_PopulatedPkFieldShouldErr(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.ProtoMarshallerMock{}

	topicMessage := &data.TopicMessage{
		Version:   libp2p.CurrentTopicMessageVersion,
		Timestamp: time.Now().Unix(),
		Payload:   []byte("data"),
		Pk:        []byte("p"),
	}
	buff, _ := marshalizer.Marshal(topicMessage)
	topic := "topic"
	mes := &pb.Message{
		From:  getRandomID(),
		Data:  buff,
		Topic: &topic,
	}

	pMes := &pubsub.Message{Message: mes}
	m, err := libp2p.NewMessage(pMes, marshalizer)

	assert.True(t, check.IfNil(m))
	assert.True(t, errors.Is(err, p2p.ErrUnsupportedFields))
}

func TestMessage_PopulatedSigFieldShouldErr(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.ProtoMarshallerMock{}

	topicMessage := &data.TopicMessage{
		Version:        libp2p.CurrentTopicMessageVersion,
		Timestamp:      time.Now().Unix(),
		Payload:        []byte("data"),
		SignatureOnPid: []byte("s"),
	}
	buff, _ := marshalizer.Marshal(topicMessage)
	topic := "topic"
	mes := &pb.Message{
		From:  getRandomID(),
		Data:  buff,
		Topic: &topic,
	}

	pMes := &pubsub.Message{Message: mes}
	m, err := libp2p.NewMessage(pMes, marshalizer)

	assert.True(t, check.IfNil(m))
	assert.True(t, errors.Is(err, p2p.ErrUnsupportedFields))
}

func TestMessage_NilTopic(t *testing.T) {
	t.Parallel()

	id := getRandomID()
	marshalizer := &mock.ProtoMarshallerMock{}

	topicMessage := &data.TopicMessage{
		Version:   libp2p.CurrentTopicMessageVersion,
		Timestamp: time.Now().Unix(),
		Payload:   []byte("data"),
	}
	buff, _ := marshalizer.Marshal(topicMessage)
	mes := &pb.Message{
		From:  id,
		Data:  buff,
		Topic: nil,
	}
	pMes := &pubsub.Message{Message: mes}
	m, err := libp2p.NewMessage(pMes, marshalizer)

	assert.Equal(t, p2p.ErrNilTopic, err)
	assert.True(t, check.IfNil(m))
}

func TestMessage_NilMessage(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.ProtoMarshallerMock{}

	m, err := libp2p.NewMessage(nil, marshalizer)

	assert.Equal(t, p2p.ErrNilMessage, err)
	assert.True(t, check.IfNil(m))
}
