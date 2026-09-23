package gitops

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ErrInvalidPath is returned when a file path cannot be mapped to valid
// namespace/service coordinates.
var ErrInvalidPath = errors.New("invalid secret path")

// ErrPathOutsideRoot is returned when a file path is not under the configured
// root at all (rather than under the root but shaped wrong). It is a distinct
// sentinel — always wrapped together with ErrInvalidPath — so callers walking
// a wider tree (e.g. SyncFromPush, which sees every changed file in a repo,
// not just those under a given root) can tell "this file was never meant to
// be a secret/config" apart from "this file is under the root but doesn't
// match the path-depth convention," and only warn about the latter.
var ErrPathOutsideRoot = errors.New("path outside configured root")

// ParseSecretPath maps a repository file path to a (namespace, service, name)
// triple under the configured secrets root.
//
// Expected structure:
//
//	<secretsRoot>/<namespace>/<service>/<name>.yaml
//
// Rules:
//   - filePath must be within secretsRoot (no directory traversal)
//   - Must be exactly 3 path components after the root
//   - Must end in ".yaml"
//   - Component names must not be empty
func ParseSecretPath(secretsRoot, filePath string) (namespace, service, name string, err error) {
	// Normalise both paths to forward-slash form without trailing slashes.
	root := filepath.ToSlash(filepath.Clean(secretsRoot))
	fp := filepath.ToSlash(filepath.Clean(filePath))

	// Ensure filePath is actually within secretsRoot.
	prefix := root + "/"
	if root == "." || root == "" {
		prefix = ""
	}
	rel, ok := strings.CutPrefix(fp, prefix)
	if !ok {
		// filePath must be strictly under root (not equal to root itself).
		if fp != root {
			return "", "", "", fmt.Errorf("%w: %w: %q is not under secrets root %q", ErrPathOutsideRoot, ErrInvalidPath, filePath, secretsRoot)
		}
		return "", "", "", fmt.Errorf("%w: %w: path equals secrets root", ErrPathOutsideRoot, ErrInvalidPath)
	}

	// Reject any remaining traversal attempts.
	if strings.Contains(rel, "..") {
		return "", "", "", fmt.Errorf("%w: path traversal detected in %q", ErrInvalidPath, filePath)
	}

	// Split into components.
	parts := strings.Split(rel, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("%w: expected <namespace>/<service>/<name>.yaml, got %q", ErrInvalidPath, rel)
	}

	namespace, service, base := parts[0], parts[1], parts[2]

	if !strings.HasSuffix(base, ".yaml") {
		return "", "", "", fmt.Errorf("%w: file must end in .yaml, got %q", ErrInvalidPath, base)
	}
	name = strings.TrimSuffix(base, ".yaml")

	if namespace == "" || service == "" || name == "" {
		return "", "", "", fmt.Errorf("%w: namespace, service, and name must not be empty", ErrInvalidPath)
	}

	return namespace, service, name, nil
}

// ParseConfigPath maps a repository file path to a (namespace, service) pair
// under the configured config root.
//
// Expected structure:
//
//	<configRoot>/<namespace>/<service>.yaml
//
// Rules:
//   - filePath must be within configRoot (no directory traversal)
//   - Must be exactly 2 path components after the root
//   - Must end in ".yaml"
//   - Component names must not be empty
func ParseConfigPath(configRoot, filePath string) (namespace, service string, err error) {
	root := filepath.ToSlash(filepath.Clean(configRoot))
	fp := filepath.ToSlash(filepath.Clean(filePath))

	prefix := root + "/"
	if root == "." || root == "" {
		prefix = ""
	}
	rel, ok := strings.CutPrefix(fp, prefix)
	if !ok {
		if fp != root {
			return "", "", fmt.Errorf("%w: %w: %q is not under config root %q", ErrPathOutsideRoot, ErrInvalidPath, filePath, configRoot)
		}
		return "", "", fmt.Errorf("%w: %w: path equals config root", ErrPathOutsideRoot, ErrInvalidPath)
	}

	if strings.Contains(rel, "..") {
		return "", "", fmt.Errorf("%w: path traversal detected in %q", ErrInvalidPath, filePath)
	}

	parts := strings.Split(rel, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("%w: expected <namespace>/<service>.yaml, got %q", ErrInvalidPath, rel)
	}

	namespace, base := parts[0], parts[1]

	if !strings.HasSuffix(base, ".yaml") {
		return "", "", fmt.Errorf("%w: config file must end in .yaml, got %q", ErrInvalidPath, base)
	}
	service = strings.TrimSuffix(base, ".yaml")

	if namespace == "" || service == "" {
		return "", "", fmt.Errorf("%w: namespace and service must not be empty", ErrInvalidPath)
	}

	return namespace, service, nil
}
