package mock

import (
	"crypto/rand"

	crypto "github.com/kalyan3104/k-crypto-core-go"
	libp2pCrypto "github.com/libp2p/go-libp2p/core/crypto"
)

// privateKeyMock implements common PrivateKey interface
type privateKeyMock struct {
	privateKey libp2pCrypto.PrivKey
}

// NewPrivateKeyMock will create a new PrivateKeyMock instance
func NewPrivateKeyMock() *privateKeyMock {
	privateKey, _, _ := libp2pCrypto.GenerateSecp256k1Key(rand.Reader)

	return &privateKeyMock{
		privateKey: privateKey,
	}
}

// ToByteArray returns the byte array representation of the key
func (p *privateKeyMock) ToByteArray() ([]byte, error) {
	return p.privateKey.Raw()
}

// GeneratePublic builds a public key for the current private key
func (p *privateKeyMock) GeneratePublic() crypto.PublicKey {
	return nil
}

// Suite returns the suite used by this key
func (p *privateKeyMock) Suite() crypto.Suite {
	return nil
}

// Scalar returns the Scalar corresponding to this Private Key
func (p *privateKeyMock) Scalar() crypto.Scalar {
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (p *privateKeyMock) IsInterfaceNil() bool {
	return p == nil
}
