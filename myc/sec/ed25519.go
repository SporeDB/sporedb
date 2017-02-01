package sec

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
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

// MarshalBinary returns a PEM-armored version of this KeyRing.
func (k *KeyRingEd25519) MarshalBinary() ([]byte, error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	buf := pem.EncodeToMemory(k.armoredSecret)

	for _, key := range k.keys {
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
			b.Headers = map[string]string{
				"self": "1",
			}
		}

		raw := pem.EncodeToMemory(b)
		buf = append(buf, raw...)
	}

	return buf, nil
}

// UnmarshalBinary rebuilds a KeyRing from its PEM-armored version.
// - It may not return an error if a parse error is encountered ;
// - NewKeyRingEd25519 must be called before to instantiate the KeyRing.
func (k *KeyRingEd25519) UnmarshalBinary(data []byte) error {
	var block *pem.Block
	buffer := data

	for len(buffer) > 0 {
		block, buffer = pem.Decode(buffer)

		if block == nil {
			break
		}

		if block.Type == pemPrivateType {
			k.armoredSecret = block
			continue
		}

		if block.Type == pemPublicType {
			if block.Headers["self"] == "1" {
				_ = json.Unmarshal(block.Bytes, k.keys[""])
				continue
			}

			if block.Headers["identity"] == "" {
				continue
			}

			lvl, _ := strconv.ParseUint(block.Headers["trust"], 10, 8) // error is OK (0 means TrustNONE)

			key := &KeyEd25519{
				identity: block.Headers["identity"],
				trust:    TrustLevel(lvl),
			}
			_ = json.Unmarshal(block.Bytes, key)

			k.keys[key.identity] = key
		}
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
