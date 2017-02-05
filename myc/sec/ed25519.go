package sec

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"strconv"
	"sync"

	"golang.org/x/crypto/ed25519"
)

// KeyEd25519 is the representation of a Key for the KeyRingEd25519.
type KeyEd25519 struct {
	Public     ed25519.PublicKey
	Signatures map[string]*Signature

	identity string
	signedBy []*KeyEd25519
	trust    TrustLevel
}

// KeyRingEd25519 is a KeyRing saving data as PEM, and using the Ed25519 high-speed high-security signatures algorithm.
type KeyRingEd25519 struct {
	mutex         sync.RWMutex
	keys          map[string]*KeyEd25519
	secret        ed25519.PrivateKey
	armoredSecret *pem.Block
}

// NewKeyRingEd25519 instanciates a new KeyRingEd25519.
// It MUST be called to create a new KeyRing.
func NewKeyRingEd25519() *KeyRingEd25519 {
	return &KeyRingEd25519{
		keys: map[string]*KeyEd25519{
			"": &KeyEd25519{
				Signatures: make(map[string]*Signature),
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
	_, k.secret, err = ed25519.GenerateKey(rand.Reader)
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
		key.signedBy = nil
		copy(key.Public, data)
	}

	key.identity = identity
	key.trust = trust
	return
}

// GetPublic returns the stored public key for the provided identity.
// Providing the empty identity will return self public key.
//
// It may returns ErrKeyRingLocked or ErrUnknownIdentity.
//
// This function is thread-safe.
func (k *KeyRingEd25519) GetPublic(identity string) (data []byte, trust TrustLevel, err error) {
	if identity == "" {
		if k.Locked() {
			err = ErrKeyRingLocked
			return
		}

		data, _ = hex.DecodeString(fmt.Sprintf("%x", k.secret.Public()))
		trust = TrustULTIMATE
		return
	}

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
	k.mutex.Lock()
	defer k.mutex.Unlock()

	key, ok := k.keys[identity]
	if !ok || identity == "" {
		return
	}

	delete(k.keys, identity)

	// Remove remote signatures
	for _, signed := range k.keys {
		for i, key2 := range signed.signedBy {
			if key == key2 {
				signed.signedBy = append(signed.signedBy[:i], signed.signedBy[i+1:]...)
				break
			}
		}
	}
}

// GetSignatures returns a map of (signer, signatures) for the provided identity.
// This function is thread-safe.
func (k *KeyRingEd25519) GetSignatures(identity string) map[string]*Signature {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	key, ok := k.keys[identity]
	if !ok {
		return nil
	}

	// Copy map
	signatures := make(map[string]*Signature)
	for _, signer := range key.signedBy {
		signatures[signer.identity] = signer.Signatures[identity]
	}

	return signatures
}

// AddSignature adds a signature to the identity, from signer "from".
// If "from" equals the empty string, the KeyRing adds a new signature to the identity using its own private key.
//
// It may returns ErrKeyRingLocked or ErrUnknownIdentity.
//
// This function is thread-safe.
func (k *KeyRingEd25519) AddSignature(identity, from string, signature *Signature) error {
	k.mutex.RLock()
	key, ok := k.keys[identity]
	k.mutex.RUnlock()

	if !ok {
		return &ErrUnknownIdentity{I: identity}
	}

	var signer *KeyEd25519

	if from == "" { // local signature
		message := append(key.Public, byte(key.trust))
		signData, err := k.Sign(message)
		if err != nil {
			return err
		}

		signer = k.keys[""]
		signature = &Signature{
			Data:  signData,
			Trust: key.trust,
		}
	} else { // third-party signature
		message := append(key.Public, byte(signature.Trust))
		signer, ok = k.keys[from]
		if !ok {
			return &ErrUnknownIdentity{I: from}
		}

		err := k.Verify(from, message, signature.Data)
		if err != nil {
			return err
		}

	}

	k.mutex.Lock()
	defer k.mutex.Unlock()

	key.signedBy = append(key.signedBy, signer)
	signer.Signatures[identity] = signature
	return nil
}

// Sign signs the message with the unlocked private key.
// This function is thread-safe.
func (k *KeyRingEd25519) Sign(cleartext []byte) (signature []byte, err error) {
	if k.Locked() {
		err = ErrKeyRingLocked
		return
	}

	signature = ed25519.Sign(k.secret, cleartext)
	return
}

// Verify checks the message signed by "from".
// The addition of local trust and third-party trust levels must be greater or equals than TrustThreshold.
//
// It may returns ErrUnknownIdentity, ErrInsufficientTrust or ErrInvalidSignature.
//
// This function is thread-safe.
func (k *KeyRingEd25519) Verify(from string, cleartext, signature []byte) error {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	key, ok := k.keys[from]
	if !ok {
		return &ErrUnknownIdentity{I: from}
	}

	lvl := TrustValue[key.trust]
	for _, signer := range key.signedBy {
		a := TrustValue[signer.trust]
		b := TrustValue[signer.Signatures[from].Trust]
		if b < a {
			lvl += b
		} else {
			lvl += a
		}
	}

	if lvl < TrustThreshold {
		return &ErrInsufficientTrust{I: from}
	}

	ok = ed25519.Verify(key.Public, cleartext, signature)
	if !ok {
		return ErrInvalidSignature
	}

	return nil
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

	// Ensure self public key consistency.
	if identity == "" {
		key.Public, _, _ = k.GetPublic("")
	}

	bytes, err := json.Marshal(key)
	if err != nil {
		return nil, err
	}

	b := &pem.Block{
		Type: pemPublicType,
		Headers: map[string]string{
			"identity": key.identity,
			"trust":    fmt.Sprint(key.trust),
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
		lvl, _ := strconv.ParseUint(block.Headers["trust"], 10, 8) // error is OK (0 means TrustNONE)

		key := &KeyEd25519{
			identity: block.Headers["identity"],
			trust:    TrustLevel(lvl),
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

		if key.identity == "" {
			k.keys[""].Signatures = key.Signatures
			return
		}

		k.keys[key.identity] = key
	}

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

	// Populate signedBy slices
	for _, key := range k.keys {
		for signee := range key.Signatures {
			signeeKey, ok := k.keys[signee]
			if ok {
				signeeKey.signedBy = append(signeeKey.signedBy, key)
			}
		}
	}

	return nil
}
