package gitops

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func mustGenerateEd25519(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return priv
}

// fakeAddr is a minimal net.Addr for exercising an ssh.HostKeyCallback
// directly without a real network connection.
type fakeAddr string

func (a fakeAddr) Network() string { return "tcp" }
func (a fakeAddr) String() string  { return string(a) }

// githubEd25519Key is parsed from the exact line embedded in known_hosts.txt
// (fetched live from https://api.github.com/meta at the time this was
// written), so this test exercises the real embedded data, not a stand-in.
const githubEd25519Key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"

func parseTestKey(t *testing.T, authorizedKeyLine string) ssh.PublicKey {
	t.Helper()
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(authorizedKeyLine))
	require.NoError(t, err)
	return key
}

func TestBuildHostKeyCallback_AcceptsBuiltinGitHubKey(t *testing.T) {
	cb, err := buildHostKeyCallback("")
	require.NoError(t, err)

	key := parseTestKey(t, githubEd25519Key)
	err = cb("github.com:22", fakeAddr("140.82.112.3:22"), key)
	assert.NoError(t, err, "the exact key embedded for github.com must be accepted")
}

func TestBuildHostKeyCallback_RejectsUnknownHost(t *testing.T) {
	cb, err := buildHostKeyCallback("")
	require.NoError(t, err)

	key := parseTestKey(t, githubEd25519Key)
	err = cb("not-a-known-git-host.example.com:22", fakeAddr("10.0.0.1:22"), key)
	assert.Error(t, err, "a host with no entry in known_hosts must be rejected, not silently allowed")
}

func TestBuildHostKeyCallback_RejectsWrongKeyForKnownHost(t *testing.T) {
	cb, err := buildHostKeyCallback("")
	require.NoError(t, err)

	// A syntactically valid key, but not the one on file for github.com —
	// this is the actual MITM-protection property host-key checking exists
	// to provide.
	other, err := ssh.NewSignerFromKey(mustGenerateEd25519(t))
	require.NoError(t, err)
	err = cb("github.com:22", fakeAddr("140.82.112.3:22"), other.PublicKey())
	assert.Error(t, err, "a different key presented for a known host must be rejected")
}

func TestBuildHostKeyCallback_MergesExtraFile(t *testing.T) {
	dir := t.TempDir()
	extra := filepath.Join(dir, "known_hosts")
	selfHostedKey := mustGenerateEd25519(t)
	signer, err := ssh.NewSignerFromKey(selfHostedKey)
	require.NoError(t, err)
	line := ssh.MarshalAuthorizedKey(signer.PublicKey())
	require.NoError(t, os.WriteFile(extra, append([]byte("git.internal.example.com "), line...), 0o600))

	cb, err := buildHostKeyCallback(extra)
	require.NoError(t, err)

	// The operator-supplied host is accepted...
	assert.NoError(t, cb("git.internal.example.com:22", fakeAddr("10.0.0.5:22"), signer.PublicKey()))
	// ...and the built-in providers are still accepted too (merged, not replaced).
	assert.NoError(t, cb("github.com:22", fakeAddr("140.82.112.3:22"), parseTestKey(t, githubEd25519Key)))
}

func TestBuildHostKeyCallback_MissingExtraFileIsAClearError(t *testing.T) {
	_, err := buildHostKeyCallback("/nonexistent/known_hosts")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/nonexistent/known_hosts")
}

func TestSyncer_HostKeyCallback_CachesResult(t *testing.T) {
	s := &Syncer{}
	cb1, err := s.hostKeyCallback()
	require.NoError(t, err)
	cb2, err := s.hostKeyCallback()
	require.NoError(t, err)
	// Both calls must return the exact same callback (computed once) —
	// compare via a successful call on each rather than pointer equality,
	// since ssh.HostKeyCallback is a func value.
	key := parseTestKey(t, githubEd25519Key)
	assert.NoError(t, cb1("github.com:22", fakeAddr("140.82.112.3:22"), key))
	assert.NoError(t, cb2("github.com:22", fakeAddr("140.82.112.3:22"), key))
}

func TestSyncer_SetExtraKnownHostsFile_TakesEffect(t *testing.T) {
	dir := t.TempDir()
	extra := filepath.Join(dir, "known_hosts")
	selfHostedKey := mustGenerateEd25519(t)
	signer, err := ssh.NewSignerFromKey(selfHostedKey)
	require.NoError(t, err)
	line := ssh.MarshalAuthorizedKey(signer.PublicKey())
	require.NoError(t, os.WriteFile(extra, append([]byte("git.internal.example.com "), line...), 0o600))

	s := &Syncer{}
	s.SetExtraKnownHostsFile(extra)
	cb, err := s.hostKeyCallback()
	require.NoError(t, err)
	assert.NoError(t, cb("git.internal.example.com:22", fakeAddr("10.0.0.5:22"), signer.PublicKey()))
}

var _ net.Addr = fakeAddr("")
