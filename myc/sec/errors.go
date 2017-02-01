package sec

import "errors"

// Error messages.
var (
	ErrKeyRingLocked    = errors.New("keyring is locked")
	ErrInvalidIdentity  = errors.New("invalid identity")
	ErrInvalidPublicKey = errors.New("invalid public key")
	ErrInvalidSignature = errors.New("invalid signature")
)

// ErrUnknownIdentity is returned when an operation is asked for an unknown identity.
type ErrUnknownIdentity struct {
	I string
}

// Error returns error's string value.
func (e ErrUnknownIdentity) Error() string {
	return "unknown identity: " + e.I
}

// ErrInsufficientTrust is returned when a verification cannot be performed due to a lack of trust in one's public key.
type ErrInsufficientTrust struct {
	I string
}

// Error returns error's string value.
func (e ErrInsufficientTrust) Error() string {
	return "insufficient trust for identity: " + e.I
}
