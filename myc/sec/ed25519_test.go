package sec

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"io"
	"strings"
	"testing"

	"github.com/awnumar/memguard"
	"github.com/stretchr/testify/require"
)

const testPEMPrivateEd25519 = `-----BEGIN SPOREDB PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC,c17d3a85686217f7ad7e6a3de99a47ef

NLMYWwAQd6zfjuXVyx1OKEOjoSLp3wVXPj38NRfxCj7DwjNC0oVURVhwwb3eL/LP
5HNxbNKAGLVr5LwtqyVy6qIsd/bco31ld6gTQdXFHcw=
-----END SPOREDB PRIVATE KEY-----
`

type testKeyPairEd25519 struct {
	sec, pub string
}

var testKeyPairsEd25519 = []testKeyPairEd25519{
	{
		"f5cbfc7e538568293bb21e7cfbe9b0e91e5071e0a93b74a8721cd6f8bcd51b6562c677aefb173821269ead5e91dc6c7c888ba6a8908d2dabf21902f7d8706b5a",
		"62c677aefb173821269ead5e91dc6c7c888ba6a8908d2dabf21902f7d8706b5a",
	},
	{
		"8bb645bce494df8498687f9345ce9e9d050ff43b80c792519eb4d8a8844c4f0572acc39d3ae6c2c73e28a88c166273c97138a334b4c35eb32dddd7e95d427eb8",
		"72acc39d3ae6c2c73e28a88c166273c97138a334b4c35eb32dddd7e95d427eb8",
	},
	{
		"d45200488b20e8a215dcd06ec88b60da340a955d99aef16312c49c2c0e44da59768b9db78bdc37b98a6fe9a685af85a33a28f5ac37d4c5aea4f12881f9d13650",
		"768b9db78bdc37b98a6fe9a685af85a33a28f5ac37d4c5aea4f12881f9d13650",
	},
	{
		"2ab4b55cc6cbb333931da826643d4b08fb535aef9bf420231fab3758d30ca0c6f9d930b2aba2e83fceecd7b3e793fb01f26c66706a1f03c4b1bb39a079f0ed6f",
		"f9d930b2aba2e83fceecd7b3e793fb01f26c66706a1f03c4b1bb39a079f0ed6f",
	},
}

func getTestPubEd25519(i int) []byte {
	raw, _ := hex.DecodeString(testKeyPairsEd25519[i].pub)
	return raw
}

func getTestSecEd25519(i int) *memguard.LockedBuffer {
	raw, _ := hex.DecodeString(testKeyPairsEd25519[i].sec)
	buf, _ := memguard.NewFromBytes(raw, true)
	return buf
}

func TestEd25519(t *testing.T) {
	require.Implements(t, (*KeyRing)(nil), &KeyRingEd25519{})
}

func TestEd25519_UnlockPrivate(t *testing.T) {
	k := NewKeyRingEd25519()
	k.armoredSecret, _ = pem.Decode([]byte(testPEMPrivateEd25519))

	wrongPass, _ := memguard.NewFromBytes([]byte("wrong"), true)
	defer wrongPass.Destroy()

	rightPass, _ := memguard.NewFromBytes([]byte("password"), true)
	defer rightPass.Destroy()

	require.NotNil(t, k.UnlockPrivate(wrongPass))
	require.Nil(t, k.UnlockPrivate(rightPass))
	require.NotNil(t, k.secret)
}

func TestEd22519_CreatePrivate(t *testing.T) {
	password, _ := memguard.NewFromBytes([]byte("password"), true)
	defer password.Destroy()

	k := NewKeyRingEd25519()
	err := k.CreatePrivate(password)
	require.Nil(t, err)

	armor := string(pem.EncodeToMemory(k.armoredSecret))
	prefix := `-----BEGIN SPOREDB PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC`

	require.True(t, strings.HasPrefix(armor, prefix))
}

func TestEd25519_AddGetPublic(t *testing.T) {
	defer memguard.DestroyAll()

	k := NewKeyRingEd25519()
	k.secret = getTestSecEd25519(0)

	// Test invalid key
	err := k.AddPublic("wrong", TrustHIGH, []byte("HELLO"))
	require.Exactly(t, ErrInvalidPublicKey, err)

	// Test unknown identity
	_, _, err = k.GetPublic("wrong")
	require.NotNil(t, err)

	// Test valid key
	expected := getTestPubEd25519(1)
	require.Nil(t, k.AddPublic("a", TrustHIGH, expected))
	got, trust, err := k.GetPublic("a")
	require.Exactly(t, expected, got)
	require.Exactly(t, TrustHIGH, trust)
	require.Nil(t, err)

	// Test overwrite existing key
	require.Nil(t, k.AddPublic("a", TrustLOW, expected))
	got, trust, err = k.GetPublic("a")
	require.Nil(t, err)
	require.Exactly(t, expected, got)
	require.Exactly(t, TrustLOW, trust)

	// Test invalid identity
	require.Exactly(t, ErrInvalidIdentity, k.AddPublic("", TrustHIGH, getTestPubEd25519(0)))
}

func TestEd25519_SignVerify(t *testing.T) {
	defer memguard.DestroyAll()

	k1 := NewKeyRingEd25519()
	k1.secret = getTestSecEd25519(0)

	k2 := NewKeyRingEd25519()
	k2.stale = true
	k2.keys["k1"] = &KeyEd25519{
		Public:   getTestPubEd25519(0),
		identity: "k1",
		trust:    TrustULTIMATE,
	}

	key2 := &KeyEd25519{
		Public:   getTestPubEd25519(1),
		identity: "k2",
		trust:    TrustULTIMATE,
		Signatures: map[string]*Signature{
			"k1": &Signature{
				Trust: TrustULTIMATE,
			},
		},
	}
	k3 := &KeyRingEd25519{
		keys: map[string]*KeyEd25519{
			"k1": &KeyEd25519{
				Public:   getTestPubEd25519(0),
				identity: "k1",
				trust:    TrustNONE,
			},
			"k2": key2,
		},
		stale: true,
	}

	message := []byte("Hello World!")
	signature, err := k1.Sign(message)
	require.Nil(t, err)

	type tc struct {
		name, identity     string
		message, signature []byte
		err                bool
	}

	cases := []*tc{
		{"valid", "k1", message, signature, false},
		{"unknown", "A", message, signature, true},
		{"invalid_message", "k1", append(message, 0x00), signature, true},
		{"invalid_signature", "k1", message, append([]byte("A"), signature[1:]...), true},
		{"bad_length_signature", "k1", message, []byte("AA"), true},
	}

	for name, verifier := range map[string]*KeyRingEd25519{
		"ULTIMATE": k2,
		"PARENT":   k3,
	} {
		t.Run(name, func(t *testing.T) {
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					err := verifier.Verify(c.identity, c.message, c.signature)
					if c.err {
						require.NotNil(t, err)
					} else {
						require.Nil(t, err)
					}
				})
			}
		})
	}
}

func TestEd25519_AddGetSignature(t *testing.T) {
	defer memguard.DestroyAll()

	// Scenario: k0 will sign 2's identity and give it to k1.
	k0 := NewKeyRingEd25519()
	k0.secret = getTestSecEd25519(0)

	k1 := NewKeyRingEd25519()
	k1.secret = getTestSecEd25519(1)

	require.Nil(t, k0.AddPublic("k2", TrustHIGH, getTestPubEd25519(2)))
	require.Nil(t, k1.AddPublic("k0", TrustULTIMATE, getTestPubEd25519(0)))

	require.Nil(t, k1.GetSignatures("k2"), "not yet registered")

	require.Nil(t, k1.AddPublic("k2", TrustNONE, getTestPubEd25519(2)))
	require.Len(t, k1.GetSignatures("k2"), 0, "not yet signed by third parties")

	require.NotNil(t, k1.AddSignature("k3", "", &Signature{}), "should not accept unknown signee")
	require.NotNil(t, k1.AddSignature("k0", "k3", &Signature{}), "should not accept unknown signer")

	require.Nil(t, k1.AddSignature("k2", "", nil))
	signatures := k1.GetSignatures("k2")
	require.Len(t, signatures, 1, "expect exactly one signature")
	require.Exactly(t, TrustNONE, signatures[""].Trust)

	_ = k0.AddSignature("k2", "", nil)
	signatures = k0.GetSignatures("k2")

	s := &Signature{
		Trust: TrustULTIMATE,
		Data:  signatures[""].Data,
	}
	require.NotNil(t, k1.AddSignature("k2", "k0", s), "should not accept invalid signatures")
	require.Nil(t, k1.AddSignature("k2", "k0", signatures[""]), "should accept valid signatures")

	signatures = k1.GetSignatures("k2")
	require.Len(t, signatures, 2, "expect exactly two signatures")
	require.NotNil(t, signatures[""])
	require.NotNil(t, signatures["k0"])
}

func TestEd25519_Export(t *testing.T) {
	k := NewKeyRingEd25519()
	password, _ := memguard.NewFromBytes([]byte("password"), true)
	defer password.Destroy()

	_ = k.CreatePrivate(password)

	data, err := k.Export("")
	require.Nil(t, err)
	require.True(t, strings.HasPrefix(string(data), "-----BEGIN "))

	_, err = k.Export("unknown")
	require.NotNil(t, err)
}

func TestEd25519_Marshal(t *testing.T) {
	password, _ := memguard.NewFromBytes([]byte("password"), true)
	defer memguard.DestroyAll()

	k0 := NewKeyRingEd25519()
	k0.secret = getTestSecEd25519(0)
	k0.keys[""].Public = getTestPubEd25519(0)
	_ = k0.AddPublic("k1", TrustHIGH, getTestPubEd25519(1))
	_ = k0.AddPublic("k2", TrustLOW, getTestPubEd25519(2))
	_ = k0.AddSignature("k2", "", nil)

	k1 := NewKeyRingEd25519()
	k1.secret = getTestSecEd25519(1)
	k1.keys[""].Public = getTestPubEd25519(1)
	k1.armoredSecret, _ = x509.EncryptPEMBlock(rand.Reader, pemPrivateType, k1.secret.Buffer(), password.Buffer(), pemCipher)
	_ = k1.AddPublic("k0", TrustHIGH, getTestPubEd25519(0))
	_ = k1.AddPublic("k2", TrustNONE, getTestPubEd25519(2))
	_ = k1.AddSignature("k0", "", nil)
	_ = k1.AddSignature("k2", "k0", k0.GetSignatures("k2")[""])

	data, err := k1.MarshalBinary()
	require.Nil(t, err)
	require.True(t, strings.HasPrefix(string(data), "-----BEGIN "))
}

var armoredTestKeyRingEd25519 = []string{
	`-----BEGIN SPOREDB PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC,5403fd3d877138b7d91b0c4b7d6648ab

xo/moKPxYuhgNBxauLmL+jlJllm8x09rhNXpDFUrmipRNdkUBVsavzvXe44YgLXK
feMYN4wG8mtI72/1z/cQMhMianCLfsh2tnQxiBvsPy0=
-----END SPOREDB PRIVATE KEY-----
`, `-----BEGIN SPOREDB PUBLIC KEY-----
eyJQdWJsaWMiOiJjcXpEblRybXdzYytLS2lNRm1KenlYRTRvelMwdzE2ekxkM1g2
VjFDZnJnPSIsIlNpZ25hdHVyZXMiOnsiazAiOnsiRGF0YSI6IkVDWTdCNm5OZTE5
WFBrUXZVYzRVSVlML0NETkpEdkMrZzVsSDlidHEwYjYxUUZkK0ZjUnhzZ3FLMGVo
L2FXNnBrNG8zMVBlVWowdXVwWXNwSUkwVUFBPT0iLCJUcnVzdCI6Mn19fQ==
-----END SPOREDB PUBLIC KEY-----
`, `-----BEGIN SPOREDB PUBLIC KEY-----
identity: k0
trust: high

eyJQdWJsaWMiOiJZc1ozcnZzWE9DRW1ucTFla2R4c2ZJaUxwcWlRalMycjhoa0M5
OWh3YTFvPSIsIlNpZ25hdHVyZXMiOnsiazIiOnsiRGF0YSI6IlhhcmJHNlNoSlFX
ZlpwMmZQVkFueGpiOUFkMk5PUldWUE84Q1pNN2I0NEFPTDhCZmJIWDBwSnBENGhQ
QjFtS3ZqVUNpS3V6OFFXNjdMc3RrT1RVVEJRPT0iLCJUcnVzdCI6MX19fQ==
-----END SPOREDB PUBLIC KEY-----
`, `-----BEGIN SPOREDB PUBLIC KEY-----
-----END SPOREDB PUBLIC KEY-----
`, `-----BEGIN SPOREDB PUBLIC KEY-----
identity: k2
trust: none

eyJQdWJsaWMiOiJkb3VkdDR2Y043bUtiK21taGErRm96b285YXczMU1XdXBQRW9n
Zm5STmxBPSIsIlNpZ25hdHVyZXMiOnt9fQ==
-----END SPOREDB PUBLIC KEY-----
`, `INVALID`}

var armoredTestKeyRingEd25519Joined = strings.Join(armoredTestKeyRingEd25519, "")

func TestEd25519_Import(t *testing.T) {
	type tc struct {
		name     string
		data     int
		identity string
		trust    TrustLevel
		err      error
	}

	cases := []*tc{
		{"locally exported", 1, "k0", 1, nil},
		{"locally exported missing identity", 1, "", 1, ErrInvalidIdentity},
		{"third-party exported", 2, "k0", 1, nil},
		{"third-party exported wrong identity", 2, "k1", 1, ErrInvalidIdentity},
		{"invalid PEM", 5, "k0", 1, io.EOF},
		{"invalid JSON", 3, "k0", 1, ErrInvalidSignature},
		{"private", 0, "k0", 1, ErrInvalidIdentity},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			k := NewKeyRingEd25519()
			require.Exactly(t, c.err, k.Import([]byte(armoredTestKeyRingEd25519[c.data]), c.identity, c.trust))
		})
	}
}

func TestEd25519_Unmarshal(t *testing.T) {
	password, _ := memguard.NewFromBytes([]byte("password"), true)
	defer password.Destroy()

	k := NewKeyRingEd25519()
	require.Nil(t, k.UnmarshalBinary([]byte(armoredTestKeyRingEd25519Joined)))

	require.Nil(t, k.UnlockPrivate(password), "should retrieve correct password")

	data, trust, err := k.GetPublic("k0")
	require.Nil(t, err, "should retrieve k0's data")
	require.Exactly(t, getTestPubEd25519(0), data, "should retrive k0's public key")
	require.Exactly(t, TrustHIGH, trust, "should retrive k0's local trust level")

	signatures := k.GetSignatures("k0")
	require.NotNil(t, signatures[""], "should retrieve local signatures")
	require.Exactly(t, signatures[""].Trust, TrustLevel(0x02), "should retrieve local trust levels in signatures")

	signatures = k.GetSignatures("k2")
	require.NotNil(t, signatures["k0"], "should retrieve third-party signatures")
	require.Exactly(t, signatures["k0"].Trust, TrustLOW, "should retrieve local trust levels in third-party signatures")
}

func TestEd25519_RemovePublic(t *testing.T) {
	k := NewKeyRingEd25519()
	_ = k.UnmarshalBinary([]byte(armoredTestKeyRingEd25519Joined))

	k.RemovePublic("")
	k.RemovePublic("k3")
	k.RemovePublic("k0")

	require.NotNil(t, k.keys[""], "should not remove self key")
	require.NotNil(t, k.keys["k2"], "should not remove k2's key")
	require.Nil(t, k.keys["k0"], "must remove k0's key")

	signatures := k.GetSignatures("k2")
	require.Len(t, signatures, 0, "must remove related signatures")
}
