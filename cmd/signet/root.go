package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	flagServer    string
	flagToken     string
	flagTokenFile string
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
}

// dialAdmin opens a gRPC connection to the admin server and injects the bearer
// token into every RPC via PerRPCCredentials.
func dialAdmin(ctx context.Context) (*grpc.ClientConn, error) {
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

	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(tokenCreds{token: token}),
	)
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
type tokenCreds struct{ token string }

func (c tokenCreds) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + c.token}, nil
}

func (tokenCreds) RequireTransportSecurity() bool { return false }

func adminClient(conn *grpc.ClientConn) adminv1.AdminServiceClient {
	return adminv1.NewAdminServiceClient(conn)
}

func gitopsClient(conn *grpc.ClientConn) adminv1.GitOpsServiceClient {
	return adminv1.NewGitOpsServiceClient(conn)
}
