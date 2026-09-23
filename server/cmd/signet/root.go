package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	flagServer    string
	flagToken     string
	flagTokenFile string
	flagCACert    string
	flagForceTLS  bool
)

var rootCmd = &cobra.Command{
	Use:          "signet",
	Short:        "signet operator CLI",
	Long:         "Operator CLI for the signet secrets management service.\n\nThe admin gRPC listener is localhost-only; expose it with kubectl port-forward.",
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagServer, "server", "s", "",
		"admin gRPC server address (default: config file or localhost:8444)")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "",
		"SA bearer token for admin authentication")
	rootCmd.PersistentFlags().StringVar(&flagTokenFile, "token-file", "",
		"path to file containing SA bearer token")
	rootCmd.PersistentFlags().StringVar(&flagCACert, "ca", "",
		"path to a PEM CA certificate to trust for the admin server (implies TLS); "+
			"only meaningful for non-loopback --server addresses")
	rootCmd.PersistentFlags().BoolVar(&flagForceTLS, "tls", false,
		"use TLS even when connecting to a loopback address (TLS is always used for non-loopback addresses)")
}

// dialAdmin opens a gRPC connection to the admin server and injects the bearer
// token into every RPC via PerRPCCredentials.
//
// The admin bearer token must never be sent over a channel an on-path
// attacker could read. A loopback address (as used by the documented
// `kubectl port-forward` workflow) is trusted as-is, matching the design's
// threat model; any other address is always upgraded to TLS, using the
// system trust store or a CA supplied via --ca. This happens transparently —
// if the server's certificate cannot be verified, the TLS handshake fails
// before the token is ever sent, rather than silently falling back to
// plaintext.
func dialAdmin() (*grpc.ClientConn, error) {
	cfg, _ := readCliConfig() // best-effort; missing config → zero defaults

	addr := flagServer
	if addr == "" {
		addr = cfg.Server
	}
	if addr == "" {
		addr = "localhost:8444"
	}

	token, err := resolveToken(flagToken, flagTokenFile, cfg)
	if err != nil {
		return nil, err
	}

	creds, requireTLS, err := adminTransportCreds(addr, flagCACert, flagForceTLS)
	if err != nil {
		return nil, err
	}

	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(tokenCreds{token: token, requireTLS: requireTLS}),
	)
}

// adminTransportCreds selects transport credentials for addr. Loopback
// addresses use plaintext by default (matching the port-forward workflow);
// every other address is upgraded to TLS automatically, using caFile as the
// trusted root if given or the system trust store otherwise. forceTLS
// requests TLS even for a loopback address.
func adminTransportCreds(addr, caFile string, forceTLS bool) (creds credentials.TransportCredentials, requireTLS bool, err error) {
	host := addr
	if h, _, splitErr := net.SplitHostPort(addr); splitErr == nil {
		host = h
	}

	useTLS := forceTLS || caFile != "" || !isLoopbackHost(host)
	if !useTLS {
		return insecure.NewCredentials(), false, nil
	}

	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if caFile != "" {
		pemBytes, readErr := os.ReadFile(caFile)
		if readErr != nil {
			return nil, false, fmt.Errorf("read --ca %s: %w", caFile, readErr)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pemBytes) {
			return nil, false, fmt.Errorf("--ca %s: no PEM certificates found", caFile)
		}
		tlsCfg.RootCAs = pool
	}
	return credentials.NewTLS(tlsCfg), true, nil
}

// isLoopbackHost reports whether host is "localhost" or a loopback IP.
func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// resolveToken returns the bearer token, checking in priority order:
// flag value → flag file → config file → error.
func resolveToken(token, tokenFile string, cfg cliConfig) (string, error) {
	if token != "" {
		return strings.TrimSpace(token), nil
	}
	path := tokenFile
	if path == "" {
		path = cfg.TokenFile
	}
	if path == "" {
		return "", fmt.Errorf("no token provided: use --token or --token-file (or set token_file in %s)", "~/.config/signet/config.yaml")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read token file %s: %w", path, err)
	}
	tok := strings.TrimSpace(string(data))
	if tok == "" {
		return "", fmt.Errorf("token file %s is empty", path)
	}
	return tok, nil
}

// tokenCreds injects Authorization: Bearer <token> into every outgoing RPC.
type tokenCreds struct {
	token      string
	requireTLS bool // true once the connection is actually encrypted
}

func (c tokenCreds) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + c.token}, nil
}

func (c tokenCreds) RequireTransportSecurity() bool { return c.requireTLS }

func adminClient(conn *grpc.ClientConn) adminv1.AdminServiceClient {
	return adminv1.NewAdminServiceClient(conn)
}

func gitopsClient(conn *grpc.ClientConn) adminv1.GitOpsServiceClient {
	return adminv1.NewGitOpsServiceClient(conn)
}
