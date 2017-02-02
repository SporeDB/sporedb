package sec

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"strings"
	"testing"

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

func getTestKeyPairEd25519(t string, i int) []byte {
	data := testKeyPairsEd25519[i].pub
	if t == "sec" {
		data = testKeyPairsEd25519[i].sec
	}

	raw, _ := hex.DecodeString(data)
	return raw
}

func TestEd25519(t *testing.T) {
	require.Implements(t, (*KeyRing)(nil), &KeyRingEd25519{})
}

func TestEd25519_UnlockPrivate(t *testing.T) {
	k := NewKeyRingEd25519()
	k.armoredSecret, _ = pem.Decode([]byte(testPEMPrivateEd25519))

	require.NotNil(t, k.UnlockPrivate("wrong"))
	require.Nil(t, k.UnlockPrivate("password"))
	require.NotNil(t, k.secret)
}

func TestEd22519_CreatePrivate(t *testing.T) {
	k := NewKeyRingEd25519()
	err := k.CreatePrivate("password")
	require.Nil(t, err)

	armor := string(pem.EncodeToMemory(k.armoredSecret))
	prefix := `-----BEGIN SPOREDB PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC`

	require.True(t, strings.HasPrefix(armor, prefix))
}

func TestEd25519_AddGetPublic(t *testing.T) {
	k := NewKeyRingEd25519()
	k.secret = getTestKeyPairEd25519("sec", 0)

	// Test invalid key
	err := k.AddPublic("wrong", TrustHIGH, []byte("HELLO"))
	require.Exactly(t, ErrInvalidPublicKey, err)

	// Test unknown identity
	_, _, err = k.GetPublic("wrong")
	require.NotNil(t, err)

	// Test valid key
	expected := getTestKeyPairEd25519("pub", 1)
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
	require.Exactly(t, ErrInvalidIdentity, k.AddPublic("", TrustHIGH, getTestKeyPairEd25519("pub", 0)))

	// Test self identity
	got, trust, err = k.GetPublic("")
	require.Exactly(t, getTestKeyPairEd25519("pub", 0), got)
	require.Exactly(t, TrustULTIMATE, trust)
	require.Nil(t, err)
}

func TestEd25519_SignVerify(t *testing.T) {
	k1 := NewKeyRingEd25519()
	k1.secret = getTestKeyPairEd25519("sec", 0)

	k2 := NewKeyRingEd25519()
	k2.keys["k1"] = &KeyEd25519{
		Public:   getTestKeyPairEd25519("pub", 0),
		identity: "k1",
		trust:    TrustULTIMATE,
	}

	key2 := &KeyEd25519{
		Public:   getTestKeyPairEd25519("pub", 1),
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
				Public:   getTestKeyPairEd25519("pub", 0),
				identity: "k1",
				trust:    TrustNONE,
				signedBy: []*KeyEd25519{key2},
			},
			"k2": key2,
		},
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
	// Scenario: k0 will sign 2's identity and give it to k1.
	k0 := NewKeyRingEd25519()
	k0.secret = getTestKeyPairEd25519("sec", 0)

	k1 := NewKeyRingEd25519()
	k1.secret = getTestKeyPairEd25519("sec", 1)

	require.Nil(t, k0.AddPublic("k2", TrustHIGH, getTestKeyPairEd25519("pub", 2)))
	require.Nil(t, k1.AddPublic("k0", TrustULTIMATE, getTestKeyPairEd25519("pub", 0)))

	require.Nil(t, k1.GetSignatures("k2"), "not yet registered")

	require.Nil(t, k1.AddPublic("k2", TrustNONE, getTestKeyPairEd25519("pub", 2)))
	require.Len(t, k1.GetSignatures("k2"), 0, "not yet signed by third parties")

	require.NotNil(t, k1.AddSignature("k3", "", &Signature{}), "should not accept unknown signee")
	require.NotNil(t, k1.AddSignature("k0", "k3", &Signature{}), "should not accept unknown signer")

	require.Nil(t, k1.AddSignature("k2", "", nil))
	signatures := k1.GetSignatures("k2")
	require.NotNil(t, signatures, "expect at least one local signature")
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

func TestEd25519_Marshal(t *testing.T) {
	k0 := NewKeyRingEd25519()
	k0.secret = getTestKeyPairEd25519("sec", 0)
	_ = k0.AddPublic("k1", TrustHIGH, getTestKeyPairEd25519("pub", 1))
	_ = k0.AddPublic("k2", TrustLOW, getTestKeyPairEd25519("pub", 2))
	_ = k0.AddSignature("k2", "", nil)

	k1 := NewKeyRingEd25519()
	k1.secret = getTestKeyPairEd25519("sec", 1)
	k1.armoredSecret, _ = x509.EncryptPEMBlock(rand.Reader, pemPrivateType, k1.secret, []byte("password"), pemCipher)
	_ = k1.AddPublic("k0", TrustHIGH, getTestKeyPairEd25519("pub", 0))
	_ = k1.AddPublic("k2", TrustNONE, getTestKeyPairEd25519("pub", 2))
	_ = k1.AddSignature("k0", "", nil)
	_ = k1.AddSignature("k2", "k0", k0.GetSignatures("k2")[""])

	data, err := k1.MarshalBinary()
	require.Nil(t, err)
	require.True(t, strings.HasPrefix(string(data), "-----BEGIN "))
}

var armoredTestKeyRingEd25519 = `-----BEGIN SPOREDB PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC,1f647efb1ac2e37a5324faafd36d1db4

0w2QZhOnpMbk2PrxyjwiAkF0wz/mukhi/Q5b/0DRh7/Z4s+en/lSkGdNs4gB9PjH
3l9VO0YG1CmD5XlBcTGeehSeHEhaXsIBtppM9tp8ZEI=
-----END SPOREDB PRIVATE KEY-----
-----BEGIN SPOREDB PUBLIC KEY-----
self: 1

eyJQdWJsaWMiOm51bGwsIlNpZ25hdHVyZXMiOnsiazAiOnsiRGF0YSI6IkVDWTdC
Nm5OZTE5WFBrUXZVYzRVSVlML0NETkpEdkMrZzVsSDlidHEwYjYxUUZkK0ZjUnhz
Z3FLMGVoL2FXNnBrNG8zMVBlVWowdXVwWXNwSUkwVUFBPT0iLCJUcnVzdCI6Mn19
fQ==
-----END SPOREDB PUBLIC KEY-----
-----BEGIN SPOREDB PUBLIC KEY-----
identity: k0
trust: 2

eyJQdWJsaWMiOiJZc1ozcnZzWE9DRW1ucTFla2R4c2ZJaUxwcWlRalMycjhoa0M5
OWh3YTFvPSIsIlNpZ25hdHVyZXMiOnsiazIiOnsiRGF0YSI6IlhhcmJHNlNoSlFX
ZlpwMmZQVkFueGpiOUFkMk5PUldWUE84Q1pNN2I0NEFPTDhCZmJIWDBwSnBENGhQ
QjFtS3ZqVUNpS3V6OFFXNjdMc3RrT1RVVEJRPT0iLCJUcnVzdCI6MX19fQ==
-----END SPOREDB PUBLIC KEY-----
-----BEGIN SPOREDB PUBLIC KEY-----
identity: k2
trust: 0

eyJQdWJsaWMiOiJkb3VkdDR2Y043bUtiK21taGErRm96b285YXczMU1XdXBQRW9n
Zm5STmxBPSIsIlNpZ25hdHVyZXMiOnt9fQ==
-----END SPOREDB PUBLIC KEY-----
`

func TestEd25519_Unmarshal(t *testing.T) {
	k := NewKeyRingEd25519()
	require.Nil(t, k.UnmarshalBinary([]byte(armoredTestKeyRingEd25519)))

	require.Nil(t, k.UnlockPrivate("password"), "should retrieve correct password")

	data, trust, err := k.GetPublic("k0")
	require.Nil(t, err, "should retrieve k0's data")
	require.Exactly(t, getTestKeyPairEd25519("pub", 0), data, "should retrive k0's public key")
	require.Exactly(t, TrustHIGH, trust, "should retrive k0's local trust level")

	signatures := k.GetSignatures("k0")
	require.NotNil(t, signatures[""], "should retrieve local signatures")
	require.Exactly(t, signatures[""].Trust, TrustHIGH, "should retrieve local trust levels in signatures")

	signatures = k.GetSignatures("k2")
	require.NotNil(t, signatures["k0"], "should retrieve third-party signatures")
	require.Exactly(t, signatures["k0"].Trust, TrustLOW, "should retrieve local trust levels in third-party signatures")
}

func TestEd25519_RemovePublic(t *testing.T) {
	k := NewKeyRingEd25519()
	_ = k.UnmarshalBinary([]byte(armoredTestKeyRingEd25519))

	k.RemovePublic("")
	k.RemovePublic("k3")
	k.RemovePublic("k0")

	require.NotNil(t, k.keys[""], "should not remove self key")
	require.NotNil(t, k.keys["k2"], "should not remove k2's key")
	require.Nil(t, k.keys["k0"], "must remove k0's key")

	signatures := k.GetSignatures("k2")
	require.Len(t, signatures, 0, "must remove related signatures")
}
