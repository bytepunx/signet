package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/spf13/cobra"
)

var unsealCmd = &cobra.Command{
	Use:   "unseal",
	Short: "Unseal the signet server",
}

var unsealKeyCmd = &cobra.Command{
	Use:   "key",
	Short: "Unseal with a direct master key (direct-key mode only)",
	Long: `Read a binary master key from --key-file and submit it to the admin service.
Use this command when signet is configured for direct-key unseal (not Shamir).

The key file must contain the raw 32-byte AES-256 master key.
If you have the key as a hex string, convert it first:
  printf '%s' '<64-char-hex>' | xxd -r -p > master.key`,
	RunE: func(cmd *cobra.Command, args []string) error {
		keyFile, _ := cmd.Flags().GetString("key-file")
		if keyFile == "" {
			return fmt.Errorf("--key-file is required")
		}

		key, err := os.ReadFile(keyFile)
		if err != nil {
			return fmt.Errorf("read key file: %w", err)
		}

		ctx := cmd.Context()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := adminClient(conn).UnsealKey(ctx, &adminv1.UnsealKeyRequest{Key: key})
		if err != nil {
			return fmt.Errorf("UnsealKey: %w", err)
		}

		if resp.SharesRequired == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Server unsealed.")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Key accepted (%d/%d shares satisfied).\n",
				resp.SharesReceived, resp.SharesRequired)
		}
		return nil
	},
}

var unsealShareCmd = &cobra.Command{
	Use:   "share",
	Short: "Submit a Shamir key share (Shamir mode only)",
	Long: `Submit a single Shamir secret sharing key share to the admin service.
Once the configured threshold of shares has been received, the server unseals.

Provide the share as a hex string via --share or read it from --share-file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		shareHex, _ := cmd.Flags().GetString("share")
		shareFile, _ := cmd.Flags().GetString("share-file")

		var shareBytes []byte
		switch {
		case shareHex != "":
			b, err := hex.DecodeString(strings.TrimSpace(shareHex))
			if err != nil {
				return fmt.Errorf("decode --share hex: %w", err)
			}
			shareBytes = b
		case shareFile != "":
			data, err := os.ReadFile(shareFile)
			if err != nil {
				return fmt.Errorf("read share file: %w", err)
			}
			b, err := hex.DecodeString(strings.TrimSpace(string(data)))
			if err != nil {
				return fmt.Errorf("decode hex in %s: %w", shareFile, err)
			}
			shareBytes = b
		default:
			return fmt.Errorf("provide --share <hex> or --share-file <path>")
		}

		ctx := cmd.Context()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := adminClient(conn).UnsealShare(ctx, &adminv1.UnsealShareRequest{Share: shareBytes})
		if err != nil {
			return fmt.Errorf("UnsealShare: %w", err)
		}

		if resp.SharesReceived >= resp.SharesRequired && resp.SharesRequired > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Share accepted (%d/%d). Server is now unsealed.\n",
				resp.SharesReceived, resp.SharesRequired)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Share accepted (%d/%d).\n",
				resp.SharesReceived, resp.SharesRequired)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unsealCmd)
	unsealCmd.AddCommand(unsealKeyCmd, unsealShareCmd)

	unsealKeyCmd.Flags().String("key-file", "", "path to binary master key file (required)")

	unsealShareCmd.Flags().String("share", "", "Shamir share as a hex string")
	unsealShareCmd.Flags().String("share-file", "", "path to file containing a hex-encoded Shamir share")
}
