package main

import (
	"fmt"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show signet server seal state",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		conn, err := dialAdmin(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := adminClient(conn).Status(ctx, &adminv1.StatusRequest{})
		if err != nil {
			return fmt.Errorf("Status: %w", err)
		}

		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "State:    %s\n", stateName(resp.State))
		if resp.SharesRequired > 0 {
			fmt.Fprintf(w, "Progress: %d/%d shares received\n", resp.SharesReceived, resp.SharesRequired)
		}
		return nil
	},
}

func stateName(s adminv1.StatusResponse_State) string {
	switch s {
	case adminv1.StatusResponse_STATE_SEALED:
		return "SEALED"
	case adminv1.StatusResponse_STATE_UNSEALING:
		return "UNSEALING"
	case adminv1.StatusResponse_STATE_UNSEALED:
		return "UNSEALED"
	default:
		return "UNKNOWN"
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
