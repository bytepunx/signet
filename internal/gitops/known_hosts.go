package gitops

import (
	_ "embed"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// builtinKnownHosts covers github.com, gitlab.com, and bitbucket.org — see
// known_hosts.txt for provenance. Embedded so SSH host-key verification
// works out of the box in a minimal/distroless container with no home
// directory or shell to hold a ~/.ssh/known_hosts of its own; without this,
// go-git's default host-key callback construction fails outright ("unable
// to find any valid known_hosts file"), and GitOps sync over SSH — a
// documented core feature — never works at all.
//
//go:embed known_hosts.txt
var builtinKnownHosts []byte

// SetExtraKnownHostsFile configures an additional known_hosts-format file
// (e.g. mounted from a ConfigMap for a self-hosted git server) to merge with
// the built-in providers. Must be called, if at all, before the first repo
// sync — the resulting host-key callback is computed once and cached.
func (s *Syncer) SetExtraKnownHostsFile(path string) {
	s.extraKnownHostsFile = path
}

// hostKeyCallback returns the (cached) ssh.HostKeyCallback used to verify
// git servers' SSH host keys during deploy-key clones.
func (s *Syncer) hostKeyCallback() (ssh.HostKeyCallback, error) {
	s.hostKeyCallbackOnce.Do(func() {
		s.hostKeyCallbackVal, s.hostKeyCallbackErr = buildHostKeyCallback(s.extraKnownHostsFile)
	})
	return s.hostKeyCallbackVal, s.hostKeyCallbackErr
}

func buildHostKeyCallback(extraKnownHostsFile string) (ssh.HostKeyCallback, error) {
	// knownhosts.New only reads from files on disk (no io.Reader/[]byte
	// entry point), so the embedded content has to be materialized once.
	tmpFile, err := os.CreateTemp("", "signet-known-hosts-*")
	if err != nil {
		return nil, fmt.Errorf("write built-in known_hosts: %w", err)
	}
	defer tmpFile.Close()
	if _, err := tmpFile.Write(builtinKnownHosts); err != nil {
		return nil, fmt.Errorf("write built-in known_hosts: %w", err)
	}

	files := []string{tmpFile.Name()}
	if extraKnownHostsFile != "" {
		if _, err := os.Stat(extraKnownHostsFile); err != nil {
			return nil, fmt.Errorf("operator-supplied known_hosts file %q: %w", extraKnownHostsFile, err)
		}
		files = append(files, extraKnownHostsFile)
	}

	cb, err := knownhosts.New(files...)
	if err != nil {
		return nil, fmt.Errorf("parse known_hosts: %w", err)
	}
	return cb, nil
}
