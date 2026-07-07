package gitops

import (
	"testing"

	"filippo.io/age"
	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAgeKey_ProducesUsableKeyPair(t *testing.T) {
	keys := &mockKeys{}

	pubKey, encPrivKey, err := GenerateAgeKey(keys)
	require.NoError(t, err)
	assert.NotEmpty(t, pubKey)
	assert.Contains(t, pubKey, "age1")
	assert.NotEmpty(t, encPrivKey)
}

func TestGenerateAgeKey_TwoCallsProduceDifferentKeys(t *testing.T) {
	keys := &mockKeys{}

	pub1, _, err := GenerateAgeKey(keys)
	require.NoError(t, err)
	pub2, _, err := GenerateAgeKey(keys)
	require.NoError(t, err)

	assert.NotEqual(t, pub1, pub2)
}

func TestDecryptAgeKey_RoundtripsWithGenerateAgeKey(t *testing.T) {
	keys := &mockKeys{}

	pubKey, encPrivKey, err := GenerateAgeKey(keys)
	require.NoError(t, err)

	id, err := DecryptAgeKey(keys, pubKey, encPrivKey)
	require.NoError(t, err)
	x25519, ok := id.(*age.X25519Identity)
	require.True(t, ok, "expected *age.X25519Identity, got %T", id)
	assert.Equal(t, pubKey, x25519.Recipient().String())
}

func TestDecryptAgeKey_WrongPublicKeyFailsAuthentication(t *testing.T) {
	keys := &mockKeys{}

	pubKey, encPrivKey, err := GenerateAgeKey(keys)
	require.NoError(t, err)
	_ = pubKey

	_, err = DecryptAgeKey(keys, "age1wrongpublickey0000000000000000000000000000000000000000000000000", encPrivKey)
	require.Error(t, err)
}

func TestDecryptAgeKey_FallsBackForLegacyUnboundCiphertext(t *testing.T) {
	// Simulates an age key encrypted before AAD binding was introduced: no
	// AAD was used at encryption time, so decrypt must fall back to nil AAD
	// rather than fail outright.
	fixedKey := make([]byte, icrypto.KeySize)
	keys := &fixedKeyUnwrapper{key: fixedKey}

	plaintext := []byte("AGE-SECRET-KEY-1QQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ")
	legacyCiphertext, err := icrypto.Encrypt(fixedKey, plaintext, nil)
	require.NoError(t, err)

	// DecryptAgeKey will fail to parse this fixture as a real age identity
	// (it's not valid key material), but the point of this test is that the
	// AAD fallback itself succeeds in recovering the plaintext rather than
	// failing authentication — a real legacy key would parse successfully.
	_, err = DecryptAgeKey(keys, "age1anypublickey", legacyCiphertext)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse age identity")
}

// fixedKeyUnwrapper implements keyUnwrapper with a caller-supplied key.
type fixedKeyUnwrapper struct{ key []byte }

func (f *fixedKeyUnwrapper) Use(fn func([]byte) error) error {
	return fn(f.key)
}
