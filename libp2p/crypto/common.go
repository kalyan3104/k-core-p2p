package crypto

import (
	"fmt"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	crypto "github.com/kalyan3104/k-crypto-core-go"
	libp2pCrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ConvertPrivateKeyToLibp2pPrivateKey will convert common private key to libp2p private key
func ConvertPrivateKeyToLibp2pPrivateKey(privateKey crypto.PrivateKey) (libp2pCrypto.PrivKey, error) {
	if check.IfNil(privateKey) {
		return nil, ErrNilPrivateKey
	}

	p2pPrivateKeyBytes, err := privateKey.ToByteArray()
	if err != nil {
		return nil, err
	}

	return libp2pCrypto.UnmarshalSecp256k1PrivateKey(p2pPrivateKeyBytes)
}

// ConvertPeerIDToPublicKey will convert core peer id to common public key
func ConvertPeerIDToPublicKey(keyGen crypto.KeyGenerator, pid core.PeerID) (crypto.PublicKey, error) {
	libp2pPid, err := peer.IDFromBytes(pid.Bytes())
	if err != nil {
		return nil, err
	}

	pubk, err := libp2pPid.ExtractPublicKey()
	if err != nil {
		return nil, fmt.Errorf("cannot extract signing key: %s", err.Error())
	}

	pubKeyBytes, err := pubk.Raw()
	if err != nil {
		return nil, err
	}

	return keyGen.PublicKeyFromByteArray(pubKeyBytes)
}

// ConvertPublicKeyToPeerID will convert a public key to core.PeerID
func ConvertPublicKeyToPeerID(pk crypto.PublicKey) (core.PeerID, error) {
	if check.IfNil(pk) {
		return "", ErrNilPublicKey
	}

	pkBytes, err := pk.ToByteArray()
	if err != nil {
		return "", err
	}

	libp2pPk, err := libp2pCrypto.UnmarshalSecp256k1PublicKey(pkBytes)
	if err != nil {
		return "", err
	}

	pid, err := peer.IDFromPublicKey(libp2pPk)
	if err != nil {
		return "", err
	}

	return core.PeerID(pid), nil
}
