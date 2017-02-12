package sec

import (
	"crypto/x509"
	"encoding"
	"errors"
	"fmt"
	"strings"
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

var trustName = map[TrustLevel]string{
	TrustNONE:     "none",
	TrustLOW:      "low",
	TrustHIGH:     "high",
	TrustULTIMATE: "ultimate",
}

// ParseTrust returns a TrustLevel from its string representation.
func ParseTrust(trust string) (TrustLevel, error) {
	trust = strings.ToLower(trust)
	for lvl, str := range trustName {
		if str == trust {
			return lvl, nil
		}
	}

	return TrustNONE, errors.New("unrecognized trust level")
}

func (t TrustLevel) String() string {
	return trustName[t]
}

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
	ListPublic() []ListedKey
	GetPublic(identity string) (data []byte, trust TrustLevel, err error)
	RemovePublic(identity string)
	GetSignatures(identity string) map[string]*Signature
	AddSignature(identity, from string, signature *Signature) error
	Verify(from string, cleartext, signature []byte) (err error)
	Trusted(identity string) error
}

// Exporter shall export a particular credential or a whole set.
type Exporter interface {
	encoding.BinaryMarshaler
	Export(identity string) ([]byte, error)
}

// Importer shall import a particular credential or a whole set.
type Importer interface {
	encoding.BinaryUnmarshaler
	Import(data []byte, identity string, trust TrustLevel) error
}

// KeyRing shall store private and public keys while providing cryptographic functions.
type KeyRing interface {
	PrivateKeyHolder
	PublicKeysHolder
	Exporter
	Importer
}

// ListedKey shall contain one function returning basic informations about one's key.
type ListedKey interface {
	Info() (identity string, data []byte, trust TrustLevel)
}

// ByIdentity is a helper to sort ListeKey by their identity.
type ByIdentity []ListedKey

func (a ByIdentity) Len() int      { return len(a) }
func (a ByIdentity) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByIdentity) Less(i, j int) bool {
	ii, _, _ := a[i].Info()
	jj, _, _ := a[j].Info()
	return ii < jj
}

// Signature represents a local or third-party public key's signature.
type Signature struct {
	Data  []byte
	Trust TrustLevel
}

// Fingerprint is a helper function to get a human-friendly representation of one's key.
func Fingerprint(data []byte) string {
	if len(data) < 4 {
		return ""
	}

	return strings.Replace(fmt.Sprintf("% X", data[len(data)-5:]), " ", ":", -1)
}
