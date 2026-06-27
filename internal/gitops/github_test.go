package gitops

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	gogithub "github.com/google/go-github/v68/github"
)

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	secret := []byte("webhook-secret")
	payload := []byte(`{"ref":"refs/heads/main"}`)

	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if err := VerifyWebhookSignature(payload, sig, secret); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVerifyWebhookSignature_WrongSecret(t *testing.T) {
	payload := []byte(`{"ref":"refs/heads/main"}`)
	mac := hmac.New(sha256.New, []byte("correct-secret"))
	mac.Write(payload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	err := VerifyWebhookSignature(payload, sig, []byte("wrong-secret"))
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyWebhookSignature_MissingPrefix(t *testing.T) {
	err := VerifyWebhookSignature([]byte("payload"), "abcdef", []byte("secret"))
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature for missing prefix, got %v", err)
	}
}

func TestVerifyWebhookSignature_MalformedHex(t *testing.T) {
	err := VerifyWebhookSignature([]byte("payload"), "sha256=not-hex!!", []byte("secret"))
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature for bad hex, got %v", err)
	}
}

func TestParsePushEvent(t *testing.T) {
	ref := "refs/heads/main"
	after := "abc123"
	repoURL := "https://github.com/org/repo"
	payload, _ := json.Marshal(map[string]interface{}{
		"ref":   ref,
		"after": after,
		"repository": map[string]interface{}{
			"clone_url": repoURL,
		},
		"commits": []map[string]interface{}{
			{"added": []string{"secrets/ns/svc/key.yaml"}},
		},
	})

	event, err := ParsePushEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gogithub.Stringify(event.Ref) != fmt.Sprintf("%q", ref) {
		t.Errorf("Ref = %v, want %q", event.Ref, ref)
	}
}

func TestParsePushEvent_InvalidJSON(t *testing.T) {
	_, err := ParsePushEvent([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestBranchFromRef(t *testing.T) {
	cases := []struct{ ref, want string }{
		{"refs/heads/main", "main"},
		{"refs/heads/feature/foo", "feature/foo"},
		{"main", "main"}, // non-standard, passthrough
	}
	for _, tc := range cases {
		if got := BranchFromRef(tc.ref); got != tc.want {
			t.Errorf("BranchFromRef(%q) = %q, want %q", tc.ref, got, tc.want)
		}
	}
}

func TestChangedFiles(t *testing.T) {
	str := func(s string) *string { return &s }
	event := &gogithub.PushEvent{
		Commits: []*gogithub.HeadCommit{
			{Added: []string{"secrets/ns/svc/a.yaml"}, Modified: []string{"secrets/ns/svc/b.yaml"}},
			{Removed: []string{"secrets/ns/svc/c.yaml"}, Modified: []string{"secrets/ns/svc/a.yaml"}},
		},
	}
	_ = str
	changed, deleted := ChangedFiles(event)

	changedSet := make(map[string]bool)
	for _, f := range changed {
		changedSet[f] = true
	}
	if !changedSet["secrets/ns/svc/a.yaml"] {
		t.Error("expected a.yaml in changed")
	}
	if !changedSet["secrets/ns/svc/b.yaml"] {
		t.Error("expected b.yaml in changed")
	}

	deletedSet := make(map[string]bool)
	for _, f := range deleted {
		deletedSet[f] = true
	}
	if !deletedSet["secrets/ns/svc/c.yaml"] {
		t.Error("expected c.yaml in deleted")
	}
}
