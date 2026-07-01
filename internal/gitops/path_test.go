package gitops

import (
	"errors"
	"testing"
)

func TestParseConfigPath(t *testing.T) {
	tests := []struct {
		name      string
		root      string
		path      string
		wantNS    string
		wantSvc   string
		wantErrIs error
	}{
		{
			name: "happy path",
			root: "config/", path: "config/payments/api.yaml",
			wantNS: "payments", wantSvc: "api",
		},
		{
			name: "root without trailing slash",
			root: "config", path: "config/infra/redis.yaml",
			wantNS: "infra", wantSvc: "redis",
		},
		{
			name: "nested root",
			root: "ops/config", path: "ops/config/team/svc.yaml",
			wantNS: "team", wantSvc: "svc",
		},
		{
			name: "not under root", root: "config/",
			path: "other/ns/svc.yaml", wantErrIs: ErrInvalidPath,
		},
		{
			name: "too many components — three deep is a secret, not config", root: "config/",
			path: "config/ns/svc/extra.yaml", wantErrIs: ErrInvalidPath,
		},
		{
			name: "missing service component", root: "config/",
			path: "config/ns.yaml", wantErrIs: ErrInvalidPath,
		},
		{
			name: "not yaml", root: "config/",
			path: "config/ns/svc.json", wantErrIs: ErrInvalidPath,
		},
		{
			name: "traversal attempt", root: "config/",
			path: "config/../etc/passwd.yaml", wantErrIs: ErrInvalidPath,
		},
		{
			name: "equal to root", root: "config/",
			path: "config", wantErrIs: ErrInvalidPath,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ns, svc, err := ParseConfigPath(tc.root, tc.path)
			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Errorf("got err %v, want %v", err, tc.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ns != tc.wantNS || svc != tc.wantSvc {
				t.Errorf("got (%q, %q), want (%q, %q)", ns, svc, tc.wantNS, tc.wantSvc)
			}
		})
	}
}

func TestParseSecretPath(t *testing.T) {
	tests := []struct {
		name      string
		root      string
		path      string
		wantNS    string
		wantSvc   string
		wantName  string
		wantErrIs error
	}{
		{
			name: "happy path",
			root: "secrets/", path: "secrets/payments/api/db-password.yaml",
			wantNS: "payments", wantSvc: "api", wantName: "db-password",
		},
		{
			name: "root without trailing slash",
			root: "secrets", path: "secrets/infra/redis/password.yaml",
			wantNS: "infra", wantSvc: "redis", wantName: "password",
		},
		{
			name: "nested root",
			root: "ops/secrets", path: "ops/secrets/team/svc/key.yaml",
			wantNS: "team", wantSvc: "svc", wantName: "key",
		},
		{
			name: "not under root", root: "secrets/",
			path: "other/ns/svc/key.yaml", wantErrIs: ErrInvalidPath,
		},
		{
			name: "too few components", root: "secrets/",
			path: "secrets/ns/svc.yaml", wantErrIs: ErrInvalidPath,
		},
		{
			name: "too many components", root: "secrets/",
			path: "secrets/ns/svc/sub/key.yaml", wantErrIs: ErrInvalidPath,
		},
		{
			name: "not yaml", root: "secrets/",
			path: "secrets/ns/svc/key.json", wantErrIs: ErrInvalidPath,
		},
		{
			name: "traversal attempt", root: "secrets/",
			path: "secrets/../etc/passwd.yaml", wantErrIs: ErrInvalidPath,
		},
		{
			name: "equal to root", root: "secrets/",
			path: "secrets", wantErrIs: ErrInvalidPath,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ns, svc, name, err := ParseSecretPath(tc.root, tc.path)
			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Errorf("got err %v, want %v", err, tc.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ns != tc.wantNS || svc != tc.wantSvc || name != tc.wantName {
				t.Errorf("got (%q, %q, %q), want (%q, %q, %q)",
					ns, svc, name, tc.wantNS, tc.wantSvc, tc.wantName)
			}
		})
	}
}
