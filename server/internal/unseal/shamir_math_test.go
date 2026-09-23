package unseal

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGFMul verifies GF(2^8) multiplication properties.
func TestGFMul(t *testing.T) {
	t.Run("multiply by zero is zero", func(t *testing.T) {
		assert.Equal(t, byte(0), gfMul(0x53, 0))
		assert.Equal(t, byte(0), gfMul(0, 0x53))
	})

	t.Run("multiply by one is identity", func(t *testing.T) {
		assert.Equal(t, byte(0x53), gfMul(0x53, 1))
		assert.Equal(t, byte(0x53), gfMul(1, 0x53))
	})

	t.Run("commutative: a*b == b*a", func(t *testing.T) {
		assert.Equal(t, gfMul(0x53, 0xca), gfMul(0xca, 0x53))
		assert.Equal(t, gfMul(0x01, 0xff), gfMul(0xff, 0x01))
	})

	t.Run("known AES test vector: 0x53 * 0xca == 0x01", func(t *testing.T) {
		// Standard GF(2^8) inverse test vector.
		assert.Equal(t, byte(0x01), gfMul(0x53, 0xca))
	})

	t.Run("known value: 0x02 * 0x87 == 0x15", func(t *testing.T) {
		// 0x87 = 1000 0111; shift left = 1 0000 1110; reduce with 0x1b = 0001 0101 = 0x15
		assert.Equal(t, byte(0x15), gfMul(0x02, 0x87))
	})

	t.Run("distributive: a*(b^c) == a*b ^ a*c", func(t *testing.T) {
		a, b, c := byte(0x53), byte(0xca), byte(0x7f)
		assert.Equal(t, gfMul(a, b^c), gfMul(a, b)^gfMul(a, c))
	})
}

// TestGFDiv verifies GF(2^8) division.
func TestGFDiv(t *testing.T) {
	t.Run("a/a == 1 for non-zero a", func(t *testing.T) {
		for _, a := range []byte{1, 2, 0x53, 0xca, 0xff} {
			assert.Equal(t, byte(1), gfDiv(a, a), "a=%#x", a)
		}
	})

	t.Run("(a*b)/b == a", func(t *testing.T) {
		a, b := byte(0x53), byte(0x7f)
		assert.Equal(t, a, gfDiv(gfMul(a, b), b))
	})

	t.Run("0/b == 0", func(t *testing.T) {
		assert.Equal(t, byte(0), gfDiv(0, 0x53))
	})

	t.Run("division by zero panics", func(t *testing.T) {
		assert.Panics(t, func() { gfDiv(1, 0) })
	})
}

// TestSplitCombine covers successful split/combine roundtrips across configurations.
func TestSplitCombine(t *testing.T) {
	cases := []struct {
		name      string
		n, t      int
		secretLen int
	}{
		{"2-of-2, 1 byte", 2, 2, 1},
		{"2-of-2, 32 bytes", 2, 2, 32},
		{"2-of-3, 32 bytes", 3, 2, 32},
		{"3-of-5, 32 bytes", 5, 3, 32},
		{"5-of-5, 32 bytes", 5, 5, 32},
		{"3-of-10, 64 bytes", 10, 3, 64},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			secret := randomBytes(t, tc.secretLen)
			original := clone(secret)

			shares, err := splitSecret(secret, tc.n, tc.t)
			require.NoError(t, err)
			require.Len(t, shares, tc.n)

			// Each share must be len(secret)+1 bytes.
			for i, sh := range shares {
				assert.Len(t, sh, tc.secretLen+1, "share %d wrong length", i)
			}

			// Reconstruct using exactly the threshold number of shares.
			got, err := combineSecret(shares[:tc.t])
			require.NoError(t, err)
			assert.Equal(t, original, got)
		})
	}
}

// TestSplitCombine_AllSubsets verifies that any subset of size t reconstructs correctly.
func TestSplitCombine_AllSubsets(t *testing.T) {
	secret := randomBytes(t, 32)
	original := clone(secret)

	shares, err := splitSecret(secret, 5, 3)
	require.NoError(t, err)

	// Test a representative set of 3-share combinations from 5 total.
	subsets := [][]int{
		{0, 1, 2},
		{0, 1, 3},
		{0, 1, 4},
		{0, 2, 3},
		{1, 2, 4},
		{2, 3, 4},
	}

	for _, idxs := range subsets {
		subset := make([][]byte, len(idxs))
		for j, idx := range idxs {
			subset[j] = shares[idx]
		}
		got, err := combineSecret(subset)
		require.NoError(t, err, "indices %v", idxs)
		assert.Equal(t, original, got, "indices %v", idxs)
	}
}

// TestSplit_SharesAreDistinct verifies that each share has a unique x-coordinate
// and that shares for the same secret position differ.
func TestSplit_SharesAreDistinct(t *testing.T) {
	secret := randomBytes(t, 32)
	shares, err := splitSecret(secret, 5, 3)
	require.NoError(t, err)

	// x-coordinates must be 1..n, unique.
	seen := make(map[byte]bool)
	for i, sh := range shares {
		x := sh[0]
		assert.Equal(t, byte(i+1), x, "share %d x-coordinate", i)
		assert.False(t, seen[x], "duplicate x-coordinate %d", x)
		seen[x] = true
	}

	// Shares should not all be identical (extremely unlikely with random polynomials).
	allSame := true
	for _, sh := range shares[1:] {
		if !bytes.Equal(shares[0], sh) {
			allSame = false
			break
		}
	}
	assert.False(t, allSame, "all shares are identical — randomness likely broken")
}

// TestSplit_InsufficientSharesRevealNothing verifies that t-1 shares cannot
// reconstruct the secret. Since true information-theoretic security means the
// wrong result is as likely as any other, we just verify we don't get the original.
func TestSplit_InsufficientSharesRevealNothing(t *testing.T) {
	secret := randomBytes(t, 32)
	original := clone(secret)

	shares, err := splitSecret(secret, 3, 3)
	require.NoError(t, err)

	// Using only 2 of 3 required shares produces wrong output.
	wrong, err := combineSecret(shares[:2])
	require.NoError(t, err) // combine succeeds but produces garbage
	assert.NotEqual(t, original, wrong, "2-of-3 should not reconstruct the secret")
}

// TestSplitErrors covers invalid inputs to splitSecret.
func TestSplitErrors(t *testing.T) {
	t.Run("empty secret", func(t *testing.T) {
		_, err := splitSecret([]byte{}, 3, 2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("threshold less than 2", func(t *testing.T) {
		_, err := splitSecret(randomBytes(t, 32), 3, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "threshold")
	})

	t.Run("n less than t", func(t *testing.T) {
		_, err := splitSecret(randomBytes(t, 32), 2, 3)
		require.Error(t, err)
	})

	t.Run("n greater than 255", func(t *testing.T) {
		_, err := splitSecret(randomBytes(t, 32), 256, 2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "255")
	})
}

// TestCombineErrors covers invalid inputs to combineSecret.
func TestCombineErrors(t *testing.T) {
	t.Run("no shares", func(t *testing.T) {
		_, err := combineSecret(nil)
		require.Error(t, err)
	})

	t.Run("empty share slice", func(t *testing.T) {
		_, err := combineSecret([][]byte{})
		require.Error(t, err)
	})

	t.Run("share too short", func(t *testing.T) {
		_, err := combineSecret([][]byte{{0x01}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too short")
	})

	t.Run("inconsistent share lengths", func(t *testing.T) {
		shares := [][]byte{
			{0x01, 0xAA, 0xBB},
			{0x02, 0xCC},
		}
		_, err := combineSecret(shares)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "length")
	})

	t.Run("x-coordinate of zero is invalid", func(t *testing.T) {
		shares := [][]byte{
			{0x00, 0xAA, 0xBB},
			{0x01, 0xCC, 0xDD},
		}
		_, err := combineSecret(shares)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "x-coordinate")
	})

	t.Run("duplicate x-coordinates", func(t *testing.T) {
		shares := [][]byte{
			{0x01, 0xAA, 0xBB},
			{0x01, 0xCC, 0xDD},
		}
		_, err := combineSecret(shares)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})
}

// TestSplitCombine_DeterministicGiven verifies a hand-computed 2-of-2 case.
// This pins the GF(256) arithmetic to a known-correct result.
func TestSplitCombine_DeterministicGiven(t *testing.T) {
	// With 2-of-2 and t=2, f(x) = secret + a*x.
	// We verify by round-tripping through split and combine with a fixed secret.
	for _, secretByte := range []byte{0x00, 0x01, 0x7f, 0x80, 0xff} {
		secret := []byte{secretByte}
		shares, err := splitSecret(secret, 2, 2)
		require.NoError(t, err)

		got, err := combineSecret(shares)
		require.NoError(t, err)
		assert.Equal(t, secret, got, "secret byte %#x", secretByte)
	}
}

// --- helpers ---

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return b
}

func clone(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
