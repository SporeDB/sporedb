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

// PrivateKeyHolder shall be designed to safely keep one private key.
type PrivateKeyHolder interface {
	Locked() bool
	UnlockPrivate(password string) error
	CreatePrivate(password string) error
	Sign(cleartext []byte) (signature []byte, err error)
}

// PublicKeysHolder shall be designed to keep several public keys and associated signatures.
type PublicKeysHolder interface {
	AddPublic(identity string, trust TrustLevel, data []byte) error
	GetPublic(identity string) (data []byte, trust TrustLevel, err error)
	RemovePublic(identity string)
	GetSignatures(identity string) map[string]*Signature
	AddSignature(identity, from string, signature *Signature) error
	Verify(from string, cleartext, signature []byte) (err error)
}

// Exporter shall export a particular credential or a whole set.
type Exporter interface {
	encoding.BinaryMarshaler
	Export(identity string) ([]byte, error)
}

// Importer shall import a particular credential or a whole set.
type Importer interface {
	encoding.BinaryUnmarshaler
	Import(data []byte) error
}

// KeyRing shall store private and public keys while providing cryptographic functions.
type KeyRing interface {
	PrivateKeyHolder
	PublicKeysHolder
	Exporter
	Importer
}

// Signature represents a local or third-party public key's signature.
type Signature struct {
	Data  []byte
	Trust TrustLevel
}
