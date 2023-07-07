package messagecheck_test

import (
	"crypto/rand"
	"errors"
	"testing"

	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/kalyan3104/k-core-p2p/data"
	"github.com/kalyan3104/k-core-p2p/message"
	messagecheck "github.com/kalyan3104/k-core-p2p/messageCheck"
	"github.com/kalyan3104/k-core-p2p/mock"
	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	libp2pCrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func createMessageVerifierArgs() messagecheck.ArgsMessageVerifier {
	return messagecheck.ArgsMessageVerifier{
		Marshaller: &mock.MarshallerStub{},
		P2PSigner:  &mock.P2PSignerStub{},
	}
}

func TestNewMessageVerifier(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller", func(t *testing.T) {
		t.Parallel()

		args := createMessageVerifierArgs()
		args.Marshaller = nil

		mv, err := messagecheck.NewMessageVerifier(args)
		require.Nil(t, mv)
		require.Equal(t, p2p.ErrNilMarshalizer, err)
	})

	t.Run("nil p2p signer", func(t *testing.T) {
		t.Parallel()

		args := createMessageVerifierArgs()
		args.P2PSigner = nil

		mv, err := messagecheck.NewMessageVerifier(args)
		require.Nil(t, mv)
		require.Equal(t, p2p.ErrNilP2PSigner, err)
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		args := createMessageVerifierArgs()
		mv, err := messagecheck.NewMessageVerifier(args)
		require.Nil(t, err)
		require.False(t, check.IfNil(mv))
	})
}

func TestSerializeDeserialize(t *testing.T) {
	t.Parallel()

	t.Run("empty messages array", func(t *testing.T) {
		t.Parallel()

		args := createMessageVerifierArgs()
		args.Marshaller = &mock.ProtoMarshallerMock{}

		mv, err := messagecheck.NewMessageVerifier(args)
		require.Nil(t, err)

		messagesBytes, err := mv.Serialize([]p2p.MessageP2P{})
		require.Nil(t, err)

		messages, err := mv.Deserialize(messagesBytes)
		require.Nil(t, err)
		require.Equal(t, 0, len(messages))
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		args := createMessageVerifierArgs()
		args.Marshaller = &mock.ProtoMarshallerMock{}

		msgData := &data.TopicMessage{
			Version:        1,
			Payload:        []byte("payload1"),
			Timestamp:      1,
			Pk:             []byte{},
			SignatureOnPid: []byte{},
		}
		msgDataBytes, err := args.Marshaller.Marshal(msgData)
		require.Nil(t, err)

		peerID := getRandomID()

		expectedMessages := []p2p.MessageP2P{
			&message.Message{
				FromField:      peerID.Bytes(),
				PayloadField:   msgDataBytes, // it is used as data field for pubsub
				SeqNoField:     []byte("seq"),
				TopicField:     "topic",
				SignatureField: []byte("sig"),
				KeyField:       []byte("key"),
				DataField:      []byte("payload1"),
				TimestampField: 1,
				PeerField:      peerID,
			},
			&message.Message{
				FromField:      peerID.Bytes(),
				PayloadField:   msgDataBytes,
				SeqNoField:     []byte("seq"),
				TopicField:     "topic",
				SignatureField: []byte("sig"),
				KeyField:       []byte("key"),
				DataField:      []byte("payload1"),
				TimestampField: 1,
				PeerField:      peerID,
			},
		}

		mv, err := messagecheck.NewMessageVerifier(args)
		require.Nil(t, err)

		messagesBytes, err := mv.Serialize(expectedMessages)
		require.Nil(t, err)

		messages, err := mv.Deserialize(messagesBytes)
		require.Nil(t, err)

		require.Equal(t, expectedMessages, messages)
	})
}

func getRandomID() core.PeerID {
	prvKey, _, _ := libp2pCrypto.GenerateSecp256k1Key(rand.Reader)
	id, _ := peer.IDFromPublicKey(prvKey.GetPublic())

	return core.PeerID(id)
}

func TestVerify(t *testing.T) {
	t.Parallel()

	t.Run("nil p2p message", func(t *testing.T) {
		t.Parallel()

		args := createMessageVerifierArgs()
		mv, err := messagecheck.NewMessageVerifier(args)
		require.Nil(t, err)

		err = mv.Verify(nil)
		require.Equal(t, p2p.ErrNilMessage, err)
	})

	t.Run("p2p signer verify should fail", func(t *testing.T) {
		t.Parallel()

		args := createMessageVerifierArgs()

		expectedErr := errors.New("expected err")
		args.P2PSigner = &mock.P2PSignerStub{
			VerifyCalled: func(payload []byte, pid core.PeerID, signature []byte) error {
				return expectedErr
			},
		}
		mv, err := messagecheck.NewMessageVerifier(args)
		require.Nil(t, err)

		msg := &message.Message{
			FromField:      []byte("from1"),
			PayloadField:   []byte("payload1"),
			SeqNoField:     []byte("seq"),
			TopicField:     "topic",
			SignatureField: []byte("sig"),
			KeyField:       []byte("key"),
		}

		err = mv.Verify(msg)
		require.Equal(t, expectedErr, err)
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		args := createMessageVerifierArgs()

		wasCalled := false
		args.P2PSigner = &mock.P2PSignerStub{
			VerifyCalled: func(payload []byte, pid core.PeerID, signature []byte) error {
				wasCalled = true

				return nil
			},
		}
		mv, err := messagecheck.NewMessageVerifier(args)
		require.Nil(t, err)

		msg := &message.Message{
			FromField:      []byte("from1"),
			PayloadField:   []byte("payload1"),
			SeqNoField:     []byte("seq"),
			TopicField:     "topic",
			SignatureField: []byte("sig"),
			KeyField:       []byte("key"),
		}

		err = mv.Verify(msg)
		require.Nil(t, err)

		require.True(t, wasCalled)
	})
}
