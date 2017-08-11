package protocol

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"

	"golang.org/x/crypto/curve25519"

	"gitlab.com/SporeDB/sporedb/myc/sec"
)

// ecdhe provides a ecdhe encrypted session.
// See http://cr.yp.to/ecdh.html
type ecdhe struct {
	cipher.StreamReader
	cipher.StreamWriter

	KeyRing  sec.KeyRing
	Identity string

	selfPrivate [32]byte
	selfPublic  [32]byte
	peerPublic  [32]byte
	trusted     bool
}

// NewECDHESession returns a ECDHE session, using the KeyRing for
// peer authentication and signature management.
// Identity must be current node's own identity.
//
// Once Open has been called, every data passing through this Session
// will be encrypted using AES-256-CTR. Additional data authentication
// mechanism should be used for sensible informations (Raw messages for
// instance).
func NewECDHESession(kr sec.KeyRing, identity string) Session {
	return &ecdhe{
		KeyRing:  kr,
		Identity: identity,
	}
}

func (e *ecdhe) Hello() (*Hello, error) {
	// Generate private key
	_, err := rand.Read(e.selfPrivate[:])
	if err != nil {
		return nil, err
	}

	e.selfPrivate[0] &= 248
	e.selfPrivate[31] &= 127
	e.selfPrivate[31] |= 64

	// Generate public key
	curve25519.ScalarBaseMult(&(e.selfPublic), &(e.selfPrivate))

	// Build Hello message
	h := &Hello{
		Version:  Version,
		Identity: e.Identity,
		Timestamp: &timestamp.Timestamp{
			Seconds: time.Now().Unix(),
		},
		PublicKey: e.selfPublic[:],
	}

	// Sign Hello message
	raw, err := proto.Marshal(h)
	if err != nil {
		return nil, err
	}

	h.Signature, err = e.KeyRing.Sign(raw)
	return h, err
}

func (e *ecdhe) Verify(h *Hello) error {
	if h == nil {
		return ErrInvalidPublicKey
	}

	if len(h.PublicKey) != 32 {
		return ErrInvalidPublicKey
	}

	// Replay-attack protection
	if h.Timestamp.GetSeconds() < time.Now().Unix()-30 {
		return ErrOldTimestamp
	}

	// Check signature
	signature := h.Signature
	h.Signature = nil
	raw, err := proto.Marshal(h)
	if err != nil {
		return err
	}

	err = e.KeyRing.Verify(h.Identity, raw, signature)
	if err == sec.ErrInvalidSignature {
		return err
	}

	if err == nil {
		e.trusted = true
	}

	copy(e.peerPublic[:], h.PublicKey)
	return nil
}

func (e *ecdhe) Open(underlying Transport) error {
	// Get shared key
	var shared [32]byte
	curve25519.ScalarMult(&shared, &(e.selfPrivate), &(e.peerPublic))

	block, err := aes.NewCipher(shared[:])
	if err != nil {
		return err
	}

	var iv [aes.BlockSize]byte // 0-iv because the key is never the same.

	e.StreamReader = cipher.StreamReader{
		S: cipher.NewCTR(block, iv[:]),
		R: underlying,
	}

	e.StreamWriter = cipher.StreamWriter{
		S: cipher.NewCTR(block, iv[:]),
		W: underlying,
	}

	return nil
}

func (e *ecdhe) ReadByte() (byte, error) {
	d := make([]byte, 1)
	_, err := io.ReadFull(e, d)
	return d[0], err
}

func (e *ecdhe) IsTrusted() bool {
	return e.trusted
}
