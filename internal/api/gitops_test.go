package api

import (
	"strings"
	"testing"
)

// TestSkippedToErrors verifies the formatting used to surface
// bytepunx/signet#22 skips through TriggerSyncResponse/SyncBundleResponse's
// Errors field: nil in, nil out (so an empty Errors list doesn't print an
// empty "Warnings:" block on the CLI side), and one formatted message per
// skipped path otherwise.
func TestSkippedToErrors(t *testing.T) {
	if got := skippedToErrors(nil); got != nil {
		t.Errorf("skippedToErrors(nil) = %v, want nil", got)
	}
	if got := skippedToErrors([]string{}); got != nil {
		t.Errorf("skippedToErrors(empty) = %v, want nil", got)
	}

	got := skippedToErrors([]string{"config/ns/svc/extra.yaml", "secrets/ns/svc/sub/key.yaml"})
	if len(got) != 2 {
		t.Fatalf("skippedToErrors returned %d messages, want 2", len(got))
	}
	for i, path := range []string{"config/ns/svc/extra.yaml", "secrets/ns/svc/sub/key.yaml"} {
		if !strings.Contains(got[i], path) {
			t.Errorf("message %d = %q, want it to mention path %q", i, got[i], path)
		}
	}
}
