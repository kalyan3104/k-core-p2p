package crypto

import "errors"

// ErrNilPrivateKey signals that a nil private key was provided
var ErrNilPrivateKey = errors.New("nil private key")

// ErrNilPublicKey signals that a nil public key was provided
var ErrNilPublicKey = errors.New("nil public key")

// ErrNilSingleSigner signals that a nil single signer was provided
var ErrNilSingleSigner = errors.New("nil single signer")

// ErrNilKeyGenerator signals that a nil key generator was provided
var ErrNilKeyGenerator = errors.New("nil key generator")
