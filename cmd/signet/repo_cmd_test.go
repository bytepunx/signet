package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPrintSyncWarnings verifies "signet repo sync" surfaces skipped-path
// warnings (bytepunx/signet#22) instead of staying silent about them, and
// stays silent itself when there's nothing to report.
func TestPrintSyncWarnings(t *testing.T) {
	var buf bytes.Buffer
	printSyncWarnings(&buf, nil)
	assert.Empty(t, buf.String(), "no warnings should print nothing")

	buf.Reset()
	printSyncWarnings(&buf, []string{
		"skipped config/ns/svc/extra.yaml: bad depth",
		"skipped secrets/ns/svc/sub/key.yaml: bad depth",
	})
	out := buf.String()
	assert.Contains(t, out, "Warnings (2):")
	assert.Contains(t, out, "config/ns/svc/extra.yaml")
	assert.Contains(t, out, "secrets/ns/svc/sub/key.yaml")
}
