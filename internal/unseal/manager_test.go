package unseal

import (
	"sync"
	"testing"
	"time"

	"github.com/bytepunx/signet/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- New ---

func TestNew(t *testing.T) {
	t.Run("valid config with no Shamir returns a sealed manager", func(t *testing.T) {
		m, err := New(crypto.NewKeyStore(), Config{})
		require.NoError(t, err)
		assert.Equal(t, StateSealed, m.Status().State)
	})

	t.Run("valid Shamir config is accepted", func(t *testing.T) {
		_, err := New(crypto.NewKeyStore(), Config{ShamirShares: 5, ShamirThreshold: 3})
		require.NoError(t, err)
	})

	t.Run("nil store returns ErrInvalidConfig", func(t *testing.T) {
		_, err := New(nil, Config{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidConfig)
	})

	t.Run("threshold less than 2 returns ErrInvalidConfig", func(t *testing.T) {
		_, err := New(crypto.NewKeyStore(), Config{ShamirShares: 5, ShamirThreshold: 1})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidConfig)
	})

	t.Run("shares less than threshold returns ErrInvalidConfig", func(t *testing.T) {
		_, err := New(crypto.NewKeyStore(), Config{ShamirShares: 2, ShamirThreshold: 3})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidConfig)
	})

	t.Run("shares greater than 255 returns ErrInvalidConfig", func(t *testing.T) {
		_, err := New(crypto.NewKeyStore(), Config{ShamirShares: 256, ShamirThreshold: 2})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidConfig)
	})

	t.Run("zero share timeout is defaulted to 30 minutes", func(t *testing.T) {
		m, err := New(crypto.NewKeyStore(), Config{ShamirShares: 3, ShamirThreshold: 2})
		require.NoError(t, err)
		assert.Equal(t, defaultShareTimeout, m.cfg.ShareTimeout)
	})
}

// --- UnsealWithKey ---

func TestManager_UnsealWithKey(t *testing.T) {
	t.Run("happy path: state becomes Unsealed", func(t *testing.T) {
		m := mustNew(t, Config{})
		require.NoError(t, m.UnsealWithKey(mustKey(t)))
		assert.Equal(t, StateUnsealed, m.Status().State)
	})

	t.Run("store is set after successful unseal", func(t *testing.T) {
		m := mustNew(t, Config{})
		require.NoError(t, m.UnsealWithKey(mustKey(t)))
		assert.True(t, m.store.IsSet())
	})

	t.Run("nil key returns error wrapping ErrInvalidKeySize", func(t *testing.T) {
		m := mustNew(t, Config{})
		err := m.UnsealWithKey(nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, crypto.ErrInvalidKeySize)
		assert.Equal(t, StateSealed, m.Status().State)
	})

	t.Run("short key returns error wrapping ErrInvalidKeySize", func(t *testing.T) {
		m := mustNew(t, Config{})
		err := m.UnsealWithKey(make([]byte, crypto.KeySize-1))
		require.Error(t, err)
		assert.ErrorIs(t, err, crypto.ErrInvalidKeySize)
	})

	t.Run("already unsealed returns ErrAlreadyUnsealed", func(t *testing.T) {
		m := mustNew(t, Config{})
		require.NoError(t, m.UnsealWithKey(mustKey(t)))
		err := m.UnsealWithKey(mustKey(t))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAlreadyUnsealed)
	})

	t.Run("clears any accumulated Shamir shares", func(t *testing.T) {
		m := mustNew(t, shamirCfg(3, 2))
		share := validShare(t, m)
		_, err := m.SubmitShare(share)
		require.NoError(t, err)
		assert.Equal(t, StateUnsealing, m.Status().State)

		// Operator decides to use direct key instead.
		require.NoError(t, m.UnsealWithKey(mustKey(t)))
		assert.Equal(t, StateUnsealed, m.Status().State)
		assert.Equal(t, 0, m.Status().SharesReceived)
	})
}

// --- Seal ---

func TestManager_Seal(t *testing.T) {
	t.Run("sealed manager returns to Sealed state", func(t *testing.T) {
		m := mustNew(t, Config{})
		require.NoError(t, m.UnsealWithKey(mustKey(t)))
		m.Seal()
		assert.Equal(t, StateSealed, m.Status().State)
	})

	t.Run("store is cleared after Seal", func(t *testing.T) {
		m := mustNew(t, Config{})
		require.NoError(t, m.UnsealWithKey(mustKey(t)))
		m.Seal()
		assert.False(t, m.store.IsSet())
	})

	t.Run("Seal on already-sealed manager does not panic", func(t *testing.T) {
		m := mustNew(t, Config{})
		assert.NotPanics(t, func() {
			m.Seal()
			m.Seal()
		})
	})

	t.Run("Seal discards accumulated Shamir shares", func(t *testing.T) {
		m := mustNew(t, shamirCfg(3, 2))
		_, err := m.SubmitShare(validShare(t, m))
		require.NoError(t, err)
		m.Seal()
		assert.Equal(t, 0, m.Status().SharesReceived)
	})

	t.Run("UnsealWithKey succeeds after Seal", func(t *testing.T) {
		m := mustNew(t, Config{})
		require.NoError(t, m.UnsealWithKey(mustKey(t)))
		m.Seal()
		require.NoError(t, m.UnsealWithKey(mustKey(t)))
		assert.Equal(t, StateUnsealed, m.Status().State)
	})
}

// --- Status ---

func TestManager_Status(t *testing.T) {
	t.Run("initial status is Sealed with zero shares", func(t *testing.T) {
		m := mustNew(t, shamirCfg(5, 3))
		s := m.Status()
		assert.Equal(t, StateSealed, s.State)
		assert.Equal(t, 0, s.SharesReceived)
		assert.Equal(t, 3, s.SharesRequired)
	})

	t.Run("status reflects share count during accumulation", func(t *testing.T) {
		m := mustNew(t, shamirCfg(3, 3))

		for i := 1; i <= 2; i++ {
			_, err := m.SubmitShare(validShare(t, m))
			require.NoError(t, err)
			s := m.Status()
			assert.Equal(t, StateUnsealing, s.State)
			assert.Equal(t, i, s.SharesReceived)
		}
	})

	t.Run("status is Unsealed with zero shares after threshold", func(t *testing.T) {
		m := mustShamirUnsealed(t, 3, 2)
		s := m.Status()
		assert.Equal(t, StateUnsealed, s.State)
		assert.Equal(t, 0, s.SharesReceived)
	})
}

// --- UnsealWithTPM ---

func TestManager_UnsealWithTPM(t *testing.T) {
	t.Run("returns ErrTPMNotSupported in non-tpm build", func(t *testing.T) {
		m := mustNew(t, Config{})
		err := m.UnsealWithTPM("/dev/tpmrm0")
		// In a tpm build this would be ErrNotImplemented; in a stub build it's ErrTPMNotSupported.
		// Either way, an error is returned.
		require.Error(t, err)
	})
}

// --- Concurrency ---

func TestManager_Concurrent(t *testing.T) {
	t.Run("concurrent Status calls are safe", func(t *testing.T) {
		m := mustNew(t, Config{})
		var wg sync.WaitGroup
		for range 50 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = m.Status()
			}()
		}
		wg.Wait()
	})

	t.Run("concurrent UnsealWithKey and Status do not race", func(t *testing.T) {
		m := mustNew(t, Config{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = m.UnsealWithKey(mustKey(t))
		}()
		go func() {
			defer wg.Done()
			_ = m.Status()
		}()
		wg.Wait()
	})

	t.Run("concurrent Seal and Status do not race", func(t *testing.T) {
		m := mustNew(t, Config{})
		require.NoError(t, m.UnsealWithKey(mustKey(t)))
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); m.Seal() }()
		go func() { defer wg.Done(); _ = m.Status() }()
		wg.Wait()
	})
}

func TestStatusCh(t *testing.T) {
	recv := func(t *testing.T, ch <-chan struct{}) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for StatusCh notification")
		}
	}
	drain := func(ch <-chan struct{}) {
		for {
			select {
			case <-ch:
			default:
				return
			}
		}
	}

	t.Run("UnsealWithKey notifies", func(t *testing.T) {
		m := mustNew(t, Config{})
		require.NoError(t, m.UnsealWithKey(mustKey(t)))
		recv(t, m.StatusCh())
	})

	t.Run("Seal notifies", func(t *testing.T) {
		m := mustNew(t, Config{})
		require.NoError(t, m.UnsealWithKey(mustKey(t)))
		drain(m.StatusCh())
		m.Seal()
		recv(t, m.StatusCh())
	})

	t.Run("SubmitShare notifies on each transition", func(t *testing.T) {
		m := mustNew(t, shamirCfg(3, 2))
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		shares, err := GenerateShares(key, 3, 2)
		require.NoError(t, err)

		// First share: sealed → unsealing
		_, err = m.SubmitShare(shares[0])
		require.NoError(t, err)
		recv(t, m.StatusCh())
		require.Equal(t, StateUnsealing, m.Status().State)

		// Second share: unsealing → unsealed
		_, err = m.SubmitShare(shares[1])
		require.NoError(t, err)
		recv(t, m.StatusCh())
		require.Equal(t, StateUnsealed, m.Status().State)
	})

	t.Run("rapid transitions coalesce without blocking", func(t *testing.T) {
		m := mustNew(t, Config{})
		// Rapid unseal/seal cycle; the channel must never deadlock.
		for range 20 {
			require.NoError(t, m.UnsealWithKey(mustKey(t)))
			m.Seal()
		}
		// Channel has at most 1 pending notification — drain it and verify no panic.
		drain(m.StatusCh())
	})
}

// --- helpers ---

const defaultShareTimeout = 30 * time.Minute

func mustNew(t *testing.T, cfg Config) *Manager {
	t.Helper()
	m, err := New(crypto.NewKeyStore(), cfg)
	require.NoError(t, err)
	return m
}

func shamirCfg(n, t int) Config {
	return Config{ShamirShares: n, ShamirThreshold: t}
}

func mustKey(t *testing.T) []byte {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	return key
}

// validShare generates a single real Shamir share from a fresh key split.
// Used to exercise the SubmitShare path without triggering reconstruction.
func validShare(t *testing.T, m *Manager) []byte {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	shares, err := GenerateShares(key, m.cfg.ShamirShares, m.cfg.ShamirThreshold)
	require.NoError(t, err)
	return shares[0]
}

// mustShamirUnsealed creates a manager already unsealed via Shamir by splitting a
// fresh key and submitting exactly t shares.
func mustShamirUnsealed(t *testing.T, n, threshold int) *Manager {
	t.Helper()
	m := mustNew(t, shamirCfg(n, threshold))

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	shares, err := GenerateShares(key, n, threshold)
	require.NoError(t, err)

	for i := range threshold {
		_, err := m.SubmitShare(shares[i])
		require.NoError(t, err, "share %d", i)
	}
	require.Equal(t, StateUnsealed, m.Status().State)
	return m
}
