package messagecheck

import "github.com/kalyan3104/k-core/core"

type p2pSigner interface {
	Verify(payload []byte, pid core.PeerID, signature []byte) error
}
