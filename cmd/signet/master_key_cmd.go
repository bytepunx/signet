package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/spf13/cobra"
)

func init() {
	masterKeyCmd.AddCommand(masterKeyRotateCmd)
	rootCmd.AddCommand(masterKeyCmd)

	masterKeyRotateCmd.Flags().String("new-key-file", "", "path to a binary file containing the new 32-byte master key; if omitted, a random key is generated and printed once")
	masterKeyRotateCmd.Flags().Bool("yes", false, "skip the interactive confirmation prompt")
}

var masterKeyCmd = &cobra.Command{
	Use:   "master-key",
	Short: "Manage the signet master key",
}

var masterKeyRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate the master key: re-wrap every KEK (and the key-check value) under a new key",
	Long: `Re-wraps every key-encryption-key (and the key-check value) under a new
master key, then adopts it as the server's active master key. Secrets and
their DEKs are never touched directly — only their KEK layer — which is why
this operation is cheap regardless of how many secrets are stored.

This does NOT redistribute the new key to Shamir keyholders or update a
Kubernetes auto-unseal Secret. After rotating, you are responsible for:
  - Shamir mode: regenerate and redistribute new shares from the new key.
  - Kubernetes auto-unseal: update the master-key Secret with the new value
    (e.g. 'signet init --force' if using that flow), or auto-unseal will
    keep trying the old key and fail the key check on next restart.
  - Direct-key mode: securely store the new key in place of the old one.

If the in-memory key swap fails after the database has already been updated,
signetd rolls the database back to the previous wraps so the still-loaded old
key remains authoritative — but this is a narrow, best-effort safeguard, not
a substitute for verifying the server's state after rotation.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runMasterKeyRotate(cmd, cmd.InOrStdin(), cmd.OutOrStdout())
	},
}

func runMasterKeyRotate(cmd *cobra.Command, in io.Reader, out io.Writer) error {
	keyFile, _ := cmd.Flags().GetString("new-key-file")
	skipConfirm, _ := cmd.Flags().GetBool("yes")

	var newKey []byte
	var generated bool
	if keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return fmt.Errorf("read --new-key-file: %w", err)
		}
		if len(data) != 32 {
			return fmt.Errorf("--new-key-file must contain exactly 32 bytes, got %d", len(data))
		}
		newKey = data
	} else {
		newKey = make([]byte, 32)
		if _, err := rand.Read(newKey); err != nil {
			return fmt.Errorf("generate new master key: %w", err)
		}
		generated = true
	}

	warning := "WARNING: this rotates the live master key. Every KEK (and the key-check\n" +
		"value) will be re-wrapped under the new key. You are responsible for\n" +
		"redistributing the new key to Shamir keyholders or updating the\n" +
		"Kubernetes auto-unseal Secret afterward — see 'signet master-key rotate --help'."
	if err := confirmDestructive(in, out, warning, skipConfirm); err != nil {
		return err
	}

	ctx := context.Background()
	conn, err := dialAdmin()
	if err != nil {
		return err
	}
	defer conn.Close()

	resp, err := adminClient(conn).RotateMasterKey(ctx, &adminv1.RotateMasterKeyRequest{NewKey: newKey})
	if err != nil {
		return fmt.Errorf("RotateMasterKey: %w", err)
	}

	fmt.Fprintf(out, "%s\n", resp.GetMessage())
	fmt.Fprintf(out, "KEKs re-wrapped: %d\n", resp.GetKeksRewrapped())
	if generated {
		fmt.Fprintln(out, "\nNew master key (shown once — record it securely now):")
		fmt.Fprintln(out, hex.EncodeToString(newKey))
	}
	return nil
}
