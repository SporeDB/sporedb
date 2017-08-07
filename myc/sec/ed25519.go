package sec

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"sort"
	"sync"

	"golang.org/x/crypto/ed25519"
)

// KeyEd25519 is the representation of a Key for the KeyRingEd25519.
type KeyEd25519 struct {
	Public     ed25519.PublicKey
	Signatures map[string]*Signature

	identity       string
	signedBy       []*KeyEd25519
	trust          TrustLevel // set by user
	effectiveTrust TrustLevel // computed from web of trust, >= trust
}

// Info shall be used to get basic informations about this key.
func (k *KeyEd25519) Info() (identity string, data []byte, trust TrustLevel) {
	return k.identity, k.Public, k.trust
}

// KeyRingEd25519 is a KeyRing saving data as PEM, and using the Ed25519
// high-speed high-security signatures algorithm.
//
// This KeyRing also provides a lazy web of trust computation feature,
// similar to PGP's web of trust.
type KeyRingEd25519 struct {
	mutex         sync.RWMutex
	keys          map[string]*KeyEd25519
	secret        ed25519.PrivateKey
	armoredSecret *pem.Block
	stale         bool
}

// NewKeyRingEd25519 instanciates a new KeyRingEd25519.
// It MUST be called to create a new KeyRing.
func NewKeyRingEd25519() *KeyRingEd25519 {
	return &KeyRingEd25519{
		keys: map[string]*KeyEd25519{
			"": &KeyEd25519{
				trust:          TrustULTIMATE,
				effectiveTrust: TrustULTIMATE,
				Signatures:     make(map[string]*Signature),
			},
		},
	}
}

// Locked returns wether the KeyRing is currently locked or not (private key in cleartext in memory).
func (k *KeyRingEd25519) Locked() bool {
	return len(k.secret) == 0
}

// UnlockPrivate tries to decypher the private key block in memory.
func (k *KeyRingEd25519) UnlockPrivate(password string) (err error) {
	if !k.Locked() {
		return // already unlocked
	}

	k.secret, err = x509.DecryptPEMBlock(k.armoredSecret, []byte(password))
	return
}

// CreatePrivate generates a new Ed25519 private key and its associated PEM-armored block.
func (k *KeyRingEd25519) CreatePrivate(password string) (err error) {
	k.keys[""].Public, k.secret, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return
	}

	// Generate private key PEM
	k.armoredSecret, err = x509.EncryptPEMBlock(rand.Reader, pemPrivateType, k.secret, []byte(password), pemCipher)
	return
}

// AddPublic adds or overwrite a new public key in the keyring.
// It resets the related signatures if the key is modified.
//
// This function is thread-safe.
func (k *KeyRingEd25519) AddPublic(identity string, trust TrustLevel, data []byte) (err error) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	if identity == "" {
		return ErrInvalidIdentity
	}

	if len(data) != ed25519.PublicKeySize {
		return ErrInvalidPublicKey
	}

	key, ok := k.keys[identity]
	if !ok {
		key = &KeyEd25519{}
		k.keys[identity] = key
	}

	if !bytes.Equal(key.Public, data) {
		key.Public = make([]byte, ed25519.PublicKeySize)
		key.Signatures = make(map[string]*Signature)
		copy(key.Public, data)
	}

	key.identity = identity
	key.trust = trust
	k.stale = true
	return
}

// ListPublic returns every stored public key.
// The self public key is also included.
func (k *KeyRingEd25519) ListPublic() []ListedKey {
	var keys []ListedKey
	for _, key := range k.keys {
		keys = append(keys, key)
	}

	sort.Sort(ByIdentity(keys))
	return keys
}

// GetPublic returns the stored public key for the provided identity.
// Providing the empty identity will return self public key.
//
// It may returns ErrKeyRingLocked or ErrUnknownIdentity.
//
// This function is thread-safe.
func (k *KeyRingEd25519) GetPublic(identity string) (data []byte, trust TrustLevel, err error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	key, ok := k.keys[identity]
	if !ok {
		err = &ErrUnknownIdentity{I: identity}
		return
	}

	trust = key.trust
	data = make([]byte, ed25519.PublicKeySize)
	copy(data, key.Public)

	return
}

// RemovePublic removes a key from the KeyRing.
// This function is thread-safe.
func (k *KeyRingEd25519) RemovePublic(identity string) {
	if identity == "" {
		return
	}

	k.mutex.Lock()
	defer k.mutex.Unlock()
	delete(k.keys, identity)
	k.stale = true
}

// Export exports a public key to a PEM block.
func (k *KeyRingEd25519) Export(identity string) ([]byte, error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	_, ok := k.keys[identity]
	if !ok {
		return nil, &ErrUnknownIdentity{I: identity}
	}

	return k.exportUnsafe(identity)
}

func (k *KeyRingEd25519) exportUnsafe(identity string) ([]byte, error) {
	key := k.keys[identity]

	bytes, err := json.Marshal(key)
	if err != nil {
		return nil, err
	}

	b := &pem.Block{
		Type: pemPublicType,
		Headers: map[string]string{
			"identity": key.identity,
			"trust":    key.trust.String(),
		},
		Bytes: bytes,
	}

	if key.identity == "" {
		b.Headers = map[string]string{}
	}

	return pem.EncodeToMemory(b), nil
}

// MarshalBinary returns a PEM-armored version of this KeyRing.
func (k *KeyRingEd25519) MarshalBinary() ([]byte, error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	buf := pem.EncodeToMemory(k.armoredSecret)

	for identity := range k.keys {
		raw, err := k.exportUnsafe(identity)
		if err != nil {
			return nil, err
		}

		buf = append(buf, raw...)
	}

	return buf, nil
}

// Import imports a public PEM block to the keyring.
// Identity must be defined, and third-party signatures are verified afterwards.
//
// This function accepts following results of function Export:
// - Local exports (without any headears)
// - Third-party exports (with "identity" header set)
//   * If the provided identity is different that the "identity" header, an error is returned
//
// This function is thread-safe.
func (k *KeyRingEd25519) Import(data []byte, identity string, trust TrustLevel) error {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	if identity == "" {
		return ErrInvalidIdentity
	}

	_, err := k.importUnsafe(data, identity, trust)
	return err
}

func (k *KeyRingEd25519) importUnsafe(data []byte, identity string, trust TrustLevel) (remaining []byte, err error) {
	block, remaining := pem.Decode(data)

	if block == nil {
		err = io.EOF
		return
	}

	if block.Type == pemPrivateType {
		if identity != "" { // Avoid private key override when importing unsafely.
			err = ErrInvalidIdentity
			return
		}
		k.armoredSecret = block
	} else if block.Type == pemPublicType {
		lvl, _ := ParseTrust(block.Headers["trust"]) // error is handled by the default lvl value
		id := block.Headers["identity"]
		if id == "" {
			lvl = TrustULTIMATE
		}

		key := &KeyEd25519{
			identity: id,
			trust:    lvl,
		}

		err = json.Unmarshal(block.Bytes, key)
		if err != nil {
			err = ErrInvalidSignature
			return
		}

		if identity != "" {
			if key.identity != "" && key.identity != identity {
				err = ErrInvalidIdentity
				return
			}

			key.identity = identity
			key.trust = trust
		}

		k.keys[key.identity] = key
	}

	k.stale = true
	return
}

// UnmarshalBinary rebuilds a KeyRing from its PEM-armored version.
// - It may not return an error if a parse error is encountered ;
// - NewKeyRingEd25519 must be called before to instantiate the KeyRing.
func (k *KeyRingEd25519) UnmarshalBinary(data []byte) error {
	var err error
	buffer := data

	for len(buffer) > 0 && err != io.EOF {
		buffer, err = k.importUnsafe(buffer, "", 0)
	}

	return nil
}
