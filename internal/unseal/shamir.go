package unseal

import (
	"fmt"
	"time"

	"github.com/bytepunx/signet/internal/crypto"
)

// expectedShareLen is the byte length of a valid Shamir share for a master key:
// 1 byte x-coordinate + KeySize bytes of polynomial evaluation.
const expectedShareLen = 1 + crypto.KeySize

// SubmitShare accepts one Shamir share from a keyholder. Shares are accumulated
// in memory until the configured threshold is reached, at which point the master
// key is reconstructed and loaded. Returns the current Status after the share is
// processed.
//
// Errors:
//   - ErrShamirNotConfigured — manager has no ShamirThreshold set
//   - ErrAlreadyUnsealed    — server is already operational
//   - ErrSharesExpired      — previous accumulation window timed out; this share
//     starts a fresh accumulation
//   - ErrInvalidShare       — share is malformed or reconstruction failed
func (m *Manager) SubmitShare(share []byte) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cfg.ShamirThreshold == 0 {
		return m.statusLocked(), ErrShamirNotConfigured
	}

	if m.state == StateUnsealed {
		return m.statusLocked(), ErrAlreadyUnsealed
	}

	if len(share) != expectedShareLen {
		return m.statusLocked(), fmt.Errorf("%w: expected %d bytes, got %d",
			ErrInvalidShare, expectedShareLen, len(share))
	}

	// Expire stale accumulation window before accepting the new share.
	if len(m.shares) > 0 && time.Since(m.firstShareTime) > m.cfg.ShareTimeout {
		m.clearSharesLocked()
		m.state = StateSealed
		m.notifyLocked()
		return m.statusLocked(), ErrSharesExpired
	}

	// Copy the share so the caller's memory is not retained.
	shareCopy := make([]byte, len(share))
	copy(shareCopy, share)

	if len(m.shares) == 0 {
		m.firstShareTime = time.Now()
	}

	m.shares = append(m.shares, shareCopy)
	m.state = StateUnsealing
	m.notifyLocked()

	if len(m.shares) < m.cfg.ShamirThreshold {
		return m.statusLocked(), nil
	}

	// Threshold reached — attempt reconstruction.
	secret, err := combineSecret(m.shares)
	m.clearSharesLocked()

	if err != nil {
		m.state = StateSealed
		m.notifyLocked()
		return m.statusLocked(), fmt.Errorf("%w: %w", ErrInvalidShare, err)
	}

	// secret is exactly crypto.KeySize bytes (validated by splitSecret/combineSecret
	// via expectedShareLen above). store.Set zeroes secret after copying.
	if err := m.store.Set(secret); err != nil {
		m.state = StateSealed
		m.notifyLocked()
		return m.statusLocked(), fmt.Errorf("load reconstructed key: %w", err)
	}

	m.state = StateUnsealed
	m.notifyLocked()
	return m.statusLocked(), nil
}

// GenerateShares splits a master key into n Shamir shares with threshold t.
// The key slice is zeroed after splitting. Returns the shares so the operator
// can distribute them out-of-band before the key is forgotten.
//
// This is a one-time initialisation operation — callers must store the resulting
// shares securely and immediately.
func GenerateShares(key []byte, n, t int) ([][]byte, error) {
	if len(key) != crypto.KeySize {
		return nil, fmt.Errorf("generate shares: %w", crypto.ErrInvalidKeySize)
	}
	shares, err := splitSecret(key, n, t)
	if err != nil {
		return nil, fmt.Errorf("generate shares: %w", err)
	}
	// Zero the key — caller should not retain it after splitting.
	for i := range key {
		key[i] = 0
	}
	return shares, nil
}
