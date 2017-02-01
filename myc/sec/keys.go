package sec

import (
	"crypto/x509"
	"encoding"
)

// TrustLevel is a representation of a public key's trust.
type TrustLevel byte

// TrustLevel available values.
const (
	TrustNONE TrustLevel = iota
	TrustLOW
	TrustHIGH
	TrustULTIMATE
)

// TrustValue converts a TrustLevel to its value to be compared to TrustThreshold.
var TrustValue = map[TrustLevel]int{
	TrustLOW:      1,
	TrustHIGH:     2,
	TrustULTIMATE: 99,
}

// TrustThreshold is the default required TrustLevel for a verification operation.
var TrustThreshold = 2

const (
	pemPublicType  = "SPOREDB PUBLIC KEY"
	pemPrivateType = "SPOREDB PRIVATE KEY"
	pemCipher      = x509.PEMCipherAES256
)

// KeyRing shall store private and public keys while providing cryptographic functions.
type KeyRing interface {
	// Keys management functions
	Locked() bool
	UnlockPrivate(password string) error
	CreatePrivate(password string) error
	AddPublic(identity string, trust TrustLevel, data []byte) error
	GetPublic(identity string) (data []byte, trust TrustLevel, err error)
	RemovePublic(identity string)

	// Keys signature functions
	GetSignatures(identity string) map[string]*Signature
	AddSignature(identity, from string, signature *Signature) error

	// Cryptographic functions
	Sign(cleartext []byte) (signature []byte, err error)
	Verify(from string, cleartext, signature []byte) (err error)

	// Store functions
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

// Signature represents a local or third-party public key's signature.
type Signature struct {
	Data  []byte
	Trust TrustLevel
}
