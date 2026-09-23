// Package audit writes HMAC-SHA256-chained entries to the audit_log table.
// Every secret access, whether permitted or denied, must pass through Writer.Record.
// The HMAC chain allows retroactive tampering to be detected: each entry's HMAC
// covers the previous entry's HMAC concatenated with the current entry's fields.
package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/bytepunx/signet/internal/store"
)

// Writer records audit entries and maintains the HMAC chain head in memory.
// A single Writer must be shared across all callers; concurrent calls are safe.
type Writer struct {
	mu       sync.Mutex
	st       *store.Store
	chainKey []byte // HMAC key held in memory; see Zero doc comment for its lifecycle
	prevHMAC []byte // HMAC of the last written entry
}

// New creates a Writer and loads the chain head from the database.
// chainKey must be exactly 32 bytes. On an empty audit log, the chain is seeded
// with a zero vector so the first entry is still verifiable.
func New(ctx context.Context, st *store.Store, chainKey []byte) (*Writer, error) {
	if len(chainKey) != 32 {
		return nil, fmt.Errorf("audit: chain key must be 32 bytes, got %d", len(chainKey))
	}

	w := &Writer{
		st:       st,
		chainKey: chainKey,
		prevHMAC: make([]byte, 32), // zero-vector seeds the chain on first write
	}
	if err := w.loadChainHead(ctx); err != nil {
		return nil, err
	}
	return w, nil
}

// Entry is the caller-supplied audit record. HMAC is computed internally.
type Entry struct {
	SPIFFEID   string
	Action     string
	Namespace  string
	SecretName string
	Outcome    string
	PeerIP     string // optional
}

// Record computes the chained HMAC for e, writes it to the store, and advances
// the in-memory chain head. It blocks until the write is durable.
func (w *Writer) Record(ctx context.Context, e Entry) error {
	if err := validateEntry(e); err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	mac := w.computeHMAC(e, w.prevHMAC)

	if err := w.st.WriteAuditLog(ctx, &store.AuditEntry{
		SPIFFEID:   e.SPIFFEID,
		Action:     e.Action,
		Namespace:  e.Namespace,
		SecretName: e.SecretName,
		Outcome:    e.Outcome,
		PeerIP:     e.PeerIP,
		HMAC:       mac,
	}); err != nil {
		return fmt.Errorf("audit: write log: %w", err)
	}

	w.prevHMAC = mac
	return nil
}

// Zero wipes the chain key and prevHMAC from memory. Called once at process
// shutdown (not on every Seal/unseal cycle): the chain key is deliberately
// kept loaded for the whole process lifetime, including while the server is
// sealed, so that denied access attempts made against a sealed server are
// still audited. After Zero, Record will return an error on any attempt.

func (w *Writer) Zero() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for i := range w.chainKey {
		w.chainKey[i] = 0
	}
	for i := range w.prevHMAC {
		w.prevHMAC[i] = 0
	}
	w.chainKey = nil
	w.prevHMAC = nil
}

// loadChainHead fetches the HMAC of the most recently written entry so the chain
// continues correctly after a restart. Leaves prevHMAC as zero on an empty log.
func (w *Writer) loadChainHead(ctx context.Context) error {
	mac, err := w.st.LatestAuditHMAC(ctx)
	if err != nil {
		return fmt.Errorf("audit: load chain head: %w", err)
	}
	if len(mac) == 32 {
		w.prevHMAC = mac
	}
	return nil
}

// computeHMAC produces HMAC-SHA256(chainKey, prev || lenPrefix(field)...) for
// each field. Length-prefixing prevents cross-field boundary collisions.
func (w *Writer) computeHMAC(e Entry, prev []byte) []byte {
	h := hmac.New(sha256.New, w.chainKey)
	h.Write(prev)
	writeField(h, e.SPIFFEID)
	writeField(h, e.Action)
	writeField(h, e.Namespace)
	writeField(h, e.SecretName)
	writeField(h, e.Outcome)
	writeField(h, e.PeerIP)
	return h.Sum(nil)
}

type writer interface{ Write([]byte) (int, error) }

func writeField(h writer, s string) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(len(s)))
	_, _ = h.Write(buf[:])
	_, _ = h.Write([]byte(s))
}

func validateEntry(e Entry) error {
	switch {
	case e.SPIFFEID == "":
		return fmt.Errorf("audit: SPIFFEID must not be empty")
	case e.Action == "":
		return fmt.Errorf("audit: Action must not be empty")
	case e.Namespace == "":
		return fmt.Errorf("audit: Namespace must not be empty")
	case e.SecretName == "":
		return fmt.Errorf("audit: SecretName must not be empty")
	case e.Outcome == "":
		return fmt.Errorf("audit: Outcome must not be empty")
	}
	return nil
}
