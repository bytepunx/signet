package gitops

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	gogithub "github.com/google/go-github/v68/github"
)

// ErrInvalidSignature is returned when the webhook HMAC does not match.
var ErrInvalidSignature = errors.New("invalid webhook signature")

// VerifyWebhookSignature validates the GitHub X-Hub-Signature-256 header.
// sigHeader must be the full header value, e.g. "sha256=abcdef...".
// Comparison is constant-time to prevent timing attacks.
func VerifyWebhookSignature(payload []byte, sigHeader string, secret []byte) error {
	const prefix = "sha256="
	if !strings.HasPrefix(sigHeader, prefix) {
		return fmt.Errorf("%w: header must start with \"sha256=\"", ErrInvalidSignature)
	}
	gotHex := strings.TrimPrefix(sigHeader, prefix)
	got, err := hex.DecodeString(gotHex)
	if err != nil {
		return fmt.Errorf("%w: malformed signature hex: %v", ErrInvalidSignature, err)
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	want := mac.Sum(nil)

	if !hmac.Equal(got, want) {
		return ErrInvalidSignature
	}
	return nil
}

// ParsePushEvent deserialises a GitHub push webhook payload.
func ParsePushEvent(payload []byte) (*gogithub.PushEvent, error) {
	var event gogithub.PushEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("parse push event: %w", err)
	}
	return &event, nil
}

// BranchFromRef extracts the short branch name from a git ref string.
// "refs/heads/main" → "main"; other formats are returned unchanged.
func BranchFromRef(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}

// ChangedFiles collects all added, modified, and removed paths across all commits
// in a push event, deduplicated. Removals are tracked separately via the
// returned deleted slice.
func ChangedFiles(event *gogithub.PushEvent) (changed, deleted []string) {
	seen := make(map[string]bool)
	for _, commit := range event.Commits {
		for _, f := range commit.Added {
			if !seen[f] {
				changed = append(changed, f)
				seen[f] = true
			}
		}
		for _, f := range commit.Modified {
			if !seen[f] {
				changed = append(changed, f)
				seen[f] = true
			}
		}
		for _, f := range commit.Removed {
			if !seen[f] {
				deleted = append(deleted, f)
				seen[f] = true
			}
		}
	}
	// If a file was both modified and then deleted in the same push, the final
	// state is deleted; remove it from changed.
	deletedSet := make(map[string]bool, len(deleted))
	for _, f := range deleted {
		deletedSet[f] = true
	}
	filtered := changed[:0]
	for _, f := range changed {
		if !deletedSet[f] {
			filtered = append(filtered, f)
		}
	}
	return filtered, deleted
}
