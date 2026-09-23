package main

import (
	"context"
	"fmt"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/spf13/cobra"
)

func init() {
	kekCmd.AddCommand(kekRotateCmd, kekListCmd, kekPruneCmd)
	rootCmd.AddCommand(kekCmd)
}

var kekCmd = &cobra.Command{
	Use:   "kek",
	Short: "Manage the key-encryption-key (KEK) that wraps secret DEKs",
	Long: `The KEK sits between the master key and each secret's per-secret data
encryption key (DEK): Master -> KEK -> DEK -> secret. Rotating the KEK
re-wraps every secret's DEK without re-encrypting any secret's ciphertext.`,
}

var kekRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Generate a new KEK and re-wrap every secret's DEK under it",
	Long: `Generates a new key-encryption-key, deactivates the current one (retained
so DEKs not yet re-wrapped can still be decrypted), and re-wraps every
secret's DEK from the old KEK to the new one. On a fresh deployment with no
active KEK yet, this simply provisions the first one.

The old KEK is not deleted automatically — once you have confirmed the
rotation succeeded, run 'signet kek prune <old-kek-id>' to remove it.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := adminClient(conn).RotateKEK(ctx, &adminv1.RotateKEKRequest{})
		if err != nil {
			return fmt.Errorf("RotateKEK: %w", err)
		}
		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "New KEK:             %s\n", resp.GetNewKekId())
		if old := resp.GetOldKekId(); old != "" {
			fmt.Fprintf(w, "Old KEK (retained):  %s\n", old)
			fmt.Fprintf(w, "Secrets re-wrapped:  %d\n", resp.GetSecretsRewrapped())
			fmt.Fprintln(w, "Once confirmed, run 'signet kek prune "+old+"' to remove the old KEK.")
		}
		return nil
	},
}

var kekListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all KEKs (active and retained-for-decryption)",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := adminClient(conn).ListKEKs(ctx, &adminv1.ListKEKsRequest{})
		if err != nil {
			return fmt.Errorf("ListKEKs: %w", err)
		}
		if len(resp.GetKeks()) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No KEKs found. Run 'signet kek rotate' to provision one.")
			return nil
		}
		for _, k := range resp.GetKeks() {
			kekStatus := "inactive"
			if k.GetIsActive() {
				kekStatus = "active"
			}
			deact := "-"
			if d := k.GetDeactivatedAt(); d != "" {
				deact = d
			}
			fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s  created=%s  deactivated=%s\n",
				kekStatus, k.GetId(), k.GetCreatedAt(), deact)
		}
		return nil
	},
}

var kekPruneFlags struct {
	id string
}

var kekPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Permanently delete an inactive, unreferenced KEK",
	Long: `Refuses to delete the active KEK, and refuses to delete a KEK still
referenced by any secret's DEK wrap (run 'signet kek rotate' first, wait for
secrets_rewrapped to account for all secrets, then prune).`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if kekPruneFlags.id == "" {
			return fmt.Errorf("--id is required")
		}
		ctx := context.Background()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := adminClient(conn).PruneKEK(ctx, &adminv1.PruneKEKRequest{Id: kekPruneFlags.id})
		if err != nil {
			return fmt.Errorf("PruneKEK: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), resp.GetMessage())
		return nil
	},
}

func init() {
	kekPruneCmd.Flags().StringVar(&kekPruneFlags.id, "id", "", "KEK id to prune (required)")
}
