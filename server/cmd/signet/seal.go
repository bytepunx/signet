package main

import (
	"fmt"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/spf13/cobra"
)

var sealCmd = &cobra.Command{
	Use:   "seal",
	Short: "Seal the signet server (wipes master key from memory)",
	Long: `Seal the signet server, zeroing the master key from memory.
After sealing, all secret decryption requests will fail until the server is
unsealed again via 'signet unseal key' or 'signet unseal share'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		if _, err := adminClient(conn).Seal(ctx, &adminv1.SealRequest{}); err != nil {
			return fmt.Errorf("Seal: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Server sealed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sealCmd)
}
