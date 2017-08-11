package protocol

import (
	"io"
	"testing"
	"time"

	"gitlab.com/SporeDB/sporedb/myc/sec"

	"github.com/stretchr/testify/require"
)

var testKeyRingA, testKeyRingB sec.KeyRing

func init() {
	a := sec.NewKeyRingEd25519()
	_ = a.CreatePrivate("password")

	b := sec.NewKeyRingEd25519()
	_ = b.CreatePrivate("password")
	pubB, _ := b.Export("")

	_ = a.Import(pubB, "b", sec.TrustHIGH)

	testKeyRingA, testKeyRingB = a, b
}

func TestEcdhe(t *testing.T) {
	require.Implements(t, (*Session)(nil), &ecdhe{})
}

func TestEcdhe_Hello(t *testing.T) {
	e := &ecdhe{
		KeyRing:  testKeyRingA,
		Identity: "a",
	}
	h, err := e.Hello()

	require.Nil(t, err)
	require.NotZero(t, e.selfPrivate)
	require.NotZero(t, e.selfPublic)

	require.NotNil(t, h)
	require.NotNil(t, h.Signature)
	require.NotNil(t, h.PublicKey)
	require.Exactly(t, Version, h.Version)
	require.Exactly(t, "a", h.Identity)
	require.InDelta(t, time.Now().Unix(), h.Timestamp.GetSeconds(), 1)
}

func TestEcdhe_Verify(t *testing.T) {
	a := &ecdhe{
		KeyRing:  testKeyRingA,
		Identity: "a",
	}
	b := &ecdhe{
		KeyRing:  testKeyRingB,
		Identity: "b",
	}

	helloA, _ := a.Hello()
	err := b.Verify(helloA)
	require.Nil(t, err, "must accept a valid hello from an unknown sender")
	require.False(t, b.IsTrusted(), "must not trust an unknown sender")
	require.NotNil(t, b.peerPublic)

	helloB, _ := b.Hello()
	err = a.Verify(helloB)
	require.Nil(t, err, "must accept a valid hello from a known sender")
	require.True(t, a.IsTrusted(), "must trust a known & trusted sender")
	require.NotNil(t, a.peerPublic)
}

func TestEcdhe_Verify_Invalid(t *testing.T) {
	a := NewECDHESession(testKeyRingA, "a")
	b := NewECDHESession(testKeyRingB, "b")

	require.NotNil(t, a.Verify(nil), "should not crash on nil values")
	require.Exactly(t, ErrInvalidPublicKey, a.Verify(&Hello{}))

	h, _ := b.Hello()
	h.Timestamp.Seconds -= 60
	require.Exactly(t, ErrOldTimestamp, a.Verify(h), "should be resistant to replay attack")

	h, _ = b.Hello()
	h.Signature[0] = 0x00
	h.Signature[1] = 0x00
	require.Exactly(t, sec.ErrInvalidSignature, a.Verify(h))
}

func TestEcdhe_Open(t *testing.T) {
	a := NewECDHESession(testKeyRingA, "a")
	b := NewECDHESession(testKeyRingB, "b")

	helloA, _ := a.Hello()
	_ = b.Verify(helloA)
	helloB, _ := b.Hello()
	_ = a.Verify(helloB)

	transportA, transportB := NewLocalTransport()
	require.Nil(t, a.Open(transportA))
	require.Nil(t, b.Open(transportB))

	data := []byte{0x42, 0xaa, 0xbb, 0xcc, 0xdd}
	retrieved := make([]byte, len(data)-1)

	n, err := a.Write(data)
	require.Nil(t, err)
	require.Exactly(t, len(data), n)

	firstByte, err := b.ReadByte()
	require.Nil(t, err)
	require.Exactly(t, data[0], firstByte)

	n, err = io.ReadFull(b, retrieved)
	require.Nil(t, err)
	require.Exactly(t, len(retrieved), n)
	require.Exactly(t, data[1:], retrieved)

	data = make([]byte, 2050)
	data[2000] = 0x42
	retrieved = make([]byte, 2040)
	remaining := make([]byte, len(data)-len(retrieved))

	_, _ = b.Write(data)
	_, _ = io.ReadFull(a, retrieved)
	require.Exactly(t, data[:len(retrieved)], retrieved)

	n, _ = io.ReadFull(transportA, remaining)
	require.Exactly(t, len(remaining), n)
	require.NotEqual(t, make([]byte, len(remaining)), remaining, "it should encrypt data on the underlying channel")
}
