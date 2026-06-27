package unseal

import (
	"crypto/rand"
	"fmt"
)

// splitSecret splits secret into n shares requiring t to reconstruct using
// Shamir's Secret Sharing over GF(2^8).
//
// Constraints: 2 ≤ t ≤ n ≤ 255, len(secret) ≥ 1.
//
// Each returned share is len(secret)+1 bytes: the first byte is the
// x-coordinate (1-indexed), followed by the polynomial evaluations for each
// byte of the secret.
func splitSecret(secret []byte, n, t int) ([][]byte, error) {
	if len(secret) == 0 {
		return nil, fmt.Errorf("split: secret must not be empty")
	}
	if t < 2 {
		return nil, fmt.Errorf("split: threshold must be at least 2, got %d", t)
	}
	if n < t {
		return nil, fmt.Errorf("split: total shares (%d) must be >= threshold (%d)", n, t)
	}
	if n > 255 {
		return nil, fmt.Errorf("split: total shares must be <= 255, got %d", n)
	}

	shares := make([][]byte, n)
	for i := range shares {
		shares[i] = make([]byte, len(secret)+1)
		shares[i][0] = byte(i + 1) // x-coordinate is 1-indexed
	}

	coeffs := make([]byte, t-1) // random coefficients a_1 … a_{t-1}

	for byteIdx, s := range secret {
		if _, err := rand.Read(coeffs); err != nil {
			return nil, fmt.Errorf("split: generate coefficients: %w", err)
		}

		// Evaluate f(x) = s + a_1*x + a_2*x^2 + … + a_{t-1}*x^{t-1} for each share.
		for shareIdx := range n {
			x := byte(shareIdx + 1)
			y := s
			xPow := x
			for _, c := range coeffs {
				y ^= gfMul(c, xPow)
				xPow = gfMul(xPow, x)
			}
			shares[shareIdx][byteIdx+1] = y
		}
	}

	return shares, nil
}

// combineSecret reconstructs the secret from shares using Lagrange interpolation
// over GF(2^8). At least t shares (the threshold used during split) must be
// provided.
func combineSecret(shares [][]byte) ([]byte, error) {
	if len(shares) == 0 {
		return nil, fmt.Errorf("combine: no shares provided")
	}

	shareLen := len(shares[0])
	if shareLen < 2 {
		return nil, fmt.Errorf("combine: share is too short")
	}
	secretLen := shareLen - 1

	xs := make([]byte, len(shares))
	seen := make(map[byte]bool, len(shares))
	for i, sh := range shares {
		if len(sh) != shareLen {
			return nil, fmt.Errorf("combine: share %d has length %d, expected %d", i, len(sh), shareLen)
		}
		x := sh[0]
		if x == 0 {
			return nil, fmt.Errorf("combine: share %d has invalid x-coordinate 0", i)
		}
		if seen[x] {
			return nil, fmt.Errorf("combine: duplicate x-coordinate %d", x)
		}
		seen[x] = true
		xs[i] = x
	}

	secret := make([]byte, secretLen)
	for byteIdx := range secretLen {
		ys := make([]byte, len(shares))
		for i, sh := range shares {
			ys[i] = sh[byteIdx+1]
		}
		secret[byteIdx] = lagrangeAt0(xs, ys)
	}

	return secret, nil
}

// lagrangeAt0 evaluates the Lagrange interpolating polynomial at x=0 in GF(2^8).
//
//	f(0) = Σ_i y_i · Π_{j≠i} x_j / (x_i ⊕ x_j)
func lagrangeAt0(xs, ys []byte) byte {
	var result byte
	for i := range xs {
		var num, den byte = 1, 1
		for j := range xs {
			if i == j {
				continue
			}
			num = gfMul(num, xs[j])
			den = gfMul(den, xs[i]^xs[j]) // subtraction = XOR in GF(2^8)
		}
		result ^= gfMul(ys[i], gfDiv(num, den))
	}
	return result
}

// gfMul multiplies two elements in GF(2^8) using the AES reduction polynomial
// x^8 + x^4 + x^3 + x + 1 (0x11b). Implemented as Russian peasant multiplication.
func gfMul(a, b byte) byte {
	var p byte
	for range 8 {
		if b&1 != 0 {
			p ^= a
		}
		highBit := a & 0x80
		a <<= 1
		if highBit != 0 {
			a ^= 0x1b
		}
		b >>= 1
	}
	return p
}

// gfDiv divides a by b in GF(2^8) via Fermat's little theorem: b^{-1} = b^{254}.
func gfDiv(a, b byte) byte {
	if b == 0 {
		panic("unseal: division by zero in GF(2^8)")
	}
	inv := gfPow(b, 254)
	return gfMul(a, inv)
}

// gfPow computes a^n in GF(2^8).
func gfPow(a byte, n int) byte {
	result := byte(1)
	for range n {
		result = gfMul(result, a)
	}
	return result
}
