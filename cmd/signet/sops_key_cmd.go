package main

import (
	"context"
	"fmt"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/spf13/cobra"
)

func init() {
	sopsKeyCmd.AddCommand(sopsKeyGetCmd, sopsKeyRotateCmd, sopsKeyListCmd, sopsKeyPruneCmd)
	rootCmd.AddCommand(sopsKeyCmd)
}

var sopsKeyCmd = &cobra.Command{
	Use:   "sops-key",
	Short: "Manage SOPS age encryption keys",
}

var sopsKeyGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Print the currently active age public key",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		conn, err := dialAdmin(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := gitopsClient(conn).GetSOPSPublicKey(ctx, &adminv1.GetSOPSPublicKeyRequest{})
		if err != nil {
			return err
		}
		w := cmd.OutOrStdout()
		env := resp.GetEnvironment()
		if env == "" {
			env = "(global — add SIGNET_ENVIRONMENT to scope this key to an environment)"
		}
		fmt.Fprintf(w, "Public key:  %s\nFingerprint: %s\nEnvironment: %s\nCreated at:  %s\n",
			resp.GetPublicKey(), resp.GetFingerprint(), env, resp.GetCreatedAt())
		return nil
	},
}

var sopsKeyRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Generate a new age keypair and deactivate the current one",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		conn, err := dialAdmin(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := gitopsClient(conn).RotateSOPSKey(ctx, &adminv1.RotateSOPSKeyRequest{})
		if err != nil {
			return err
		}
		w := cmd.OutOrStdout()
		env := resp.GetNewEnvironment()
		if env == "" {
			env = "(global)"
		}
		fmt.Fprintf(w, "New public key:  %s\nNew fingerprint: %s\nEnvironment:     %s\n",
			resp.GetNewPublicKey(), resp.GetNewFingerprint(), env)
		if old := resp.GetOldPublicKey(); old != "" {
			fmt.Fprintf(w, "Old key retained for decryption: %s\n", old)
			fmt.Fprintln(w, "Re-encrypt your SOPS files with the new key, then run 'signet sops-key prune'.")
		}
		return nil
	},
}

var sopsKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all age keys (active and inactive)",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		conn, err := dialAdmin(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := gitopsClient(conn).ListSOPSKeys(ctx, &adminv1.ListSOPSKeysRequest{})
		if err != nil {
			return err
		}
		if len(resp.GetKeys()) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No keys found. Run 'signet sops-key rotate' to generate one.")
			return nil
		}
		for _, k := range resp.GetKeys() {
			keyStatus := "inactive"
			if k.GetIsActive() {
				keyStatus = "active"
			}
			deact := "-"
			if d := k.GetDeactivatedAt(); d != "" {
				deact = d
			}
			env := k.GetEnvironment()
			if env == "" {
				env = "(global)"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s  env=%s  fp=%s  created=%s  deactivated=%s\n",
				keyStatus, k.GetPublicKey(), env, k.GetFingerprint(), k.GetCreatedAt(), deact)
		}
		return nil
	},
}

var sopsKeyPruneFlags struct {
	publicKey string
}

var sopsKeyPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Permanently delete an inactive age key",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if sopsKeyPruneFlags.publicKey == "" {
			return fmt.Errorf("--public-key is required")
		}
		ctx := context.Background()
		conn, err := dialAdmin(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := gitopsClient(conn).PruneSOPSKey(ctx, &adminv1.PruneSOPSKeyRequest{
			PublicKey: sopsKeyPruneFlags.publicKey,
		})
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), resp.GetMessage())
		return nil
	},
}

func init() {
	sopsKeyPruneCmd.Flags().StringVar(&sopsKeyPruneFlags.publicKey, "public-key", "", "age public key to prune (required)")
}
