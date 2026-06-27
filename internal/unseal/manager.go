// Package unseal manages the sealed/unsealing/unsealed state machine and
// coordinates the three unseal mechanisms: direct key, Shamir, and TPM.
package unseal

import (
	"fmt"
	"sync"
	"time"

	"github.com/bytepunx/signet/internal/crypto"
)

// State represents the current operational state of the server.
type State int

const (
	StateSealed    State = iota // no key in memory; secrets cannot be served
	StateUnsealing              // Shamir shares accumulating; threshold not yet met
	StateUnsealed               // master key loaded; server is operational
)

func (s State) String() string {
	switch s {
	case StateSealed:
		return "sealed"
	case StateUnsealing:
		return "unsealing"
	case StateUnsealed:
		return "unsealed"
	default:
		return "unknown"
	}
}

// Config holds the configuration for the unseal Manager.
type Config struct {
	// ShamirShares is the total number of shares (n) created during the Shamir
	// split. Required when using Shamir unsealing; ignored otherwise.
	ShamirShares int

	// ShamirThreshold is the minimum number of shares (t) required to reconstruct
	// the master key. Must satisfy 2 ≤ ShamirThreshold ≤ ShamirShares.
	// Zero means Shamir unsealing is not configured.
	ShamirThreshold int

	// ShareTimeout is how long the manager will wait for all Shamir shares before
	// discarding the accumulated set. Defaults to 30 minutes if zero.
	ShareTimeout time.Duration
}

// Status reports the current state of the Manager.
type Status struct {
	State          State
	SharesReceived int
	SharesRequired int
}

// Manager coordinates the unseal state machine. It is safe for concurrent use.
type Manager struct {
	store *crypto.KeyStore
	cfg   Config

	mu             sync.Mutex
	state          State
	shares         [][]byte
	firstShareTime time.Time

	// statusCh is a capacity-1 ping channel. A value is sent (non-blocking)
	// whenever state transitions. Consumers read Status() after waking to get
	// the current state; intermediate states may be coalesced.
	statusCh chan struct{}
}

// New creates a new Manager. Returns ErrInvalidConfig if cfg is inconsistent.
func New(store *crypto.KeyStore, cfg Config) (*Manager, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: key store must not be nil", ErrInvalidConfig)
	}

	if cfg.ShamirThreshold > 0 {
		if cfg.ShamirThreshold < 2 {
			return nil, fmt.Errorf("%w: ShamirThreshold must be at least 2, got %d", ErrInvalidConfig, cfg.ShamirThreshold)
		}
		if cfg.ShamirShares < cfg.ShamirThreshold {
			return nil, fmt.Errorf("%w: ShamirShares (%d) must be >= ShamirThreshold (%d)", ErrInvalidConfig, cfg.ShamirShares, cfg.ShamirThreshold)
		}
		if cfg.ShamirShares > 255 {
			return nil, fmt.Errorf("%w: ShamirShares must be <= 255, got %d", ErrInvalidConfig, cfg.ShamirShares)
		}
	}

	if cfg.ShareTimeout == 0 {
		cfg.ShareTimeout = 30 * time.Minute
	}

	return &Manager{store: store, cfg: cfg, statusCh: make(chan struct{}, 1)}, nil
}

// StatusCh returns a channel that receives a ping (empty struct) on every state
// transition. The channel has capacity 1; rapid transitions are coalesced so
// consumers never block. Call Status() after receiving to get the current state.
func (m *Manager) StatusCh() <-chan struct{} { return m.statusCh }

// notifyLocked sends a ping to statusCh without blocking.
// Must be called with m.mu held.
func (m *Manager) notifyLocked() {
	select {
	case m.statusCh <- struct{}{}:
	default:
	}
}

// UnsealWithKey loads the master key directly into locked memory. The caller's
// key slice is zeroed. Returns ErrAlreadyUnsealed if the server is already
// operational, or a wrapped crypto error if the key is invalid.
func (m *Manager) UnsealWithKey(key []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StateUnsealed {
		return ErrAlreadyUnsealed
	}

	if err := m.store.Set(key); err != nil {
		return fmt.Errorf("unseal with key: %w", err)
	}

	m.clearSharesLocked()
	m.state = StateUnsealed
	m.notifyLocked()
	return nil
}

// Seal wipes the master key from memory and returns the server to the sealed
// state. Any accumulated Shamir shares are also discarded. Calling Seal on an
// already-sealed server is a no-op.
func (m *Manager) Seal() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store.Zero()
	m.clearSharesLocked()
	m.state = StateSealed
	m.notifyLocked()
}

// Status returns the current state of the Manager.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.statusLocked()
}

func (m *Manager) statusLocked() Status {
	return Status{
		State:          m.state,
		SharesReceived: len(m.shares),
		SharesRequired: m.cfg.ShamirThreshold,
	}
}

// clearSharesLocked zeroes and discards all accumulated Shamir shares.
// Must be called with m.mu held.
func (m *Manager) clearSharesLocked() {
	for _, sh := range m.shares {
		for i := range sh {
			sh[i] = 0
		}
	}
	m.shares = nil
	m.firstShareTime = time.Time{}
}
