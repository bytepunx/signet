package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage signet CLI configuration",
	Long: `Manage signet CLI configuration stored at ~/.config/signet/config.yaml.

Example:
  signet config set server localhost:8444
  signet config set token-file ~/.signet-token`,
}

var configSetCmd = &cobra.Command{
	Use:       "set <key> <value>",
	Short:     "Set a configuration value",
	Args:      cobra.ExactArgs(2),
	ValidArgs: []string{"server", "token-file"},
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]
		cfg, _ := readCliConfig() // missing config is fine; we'll create it
		switch key {
		case "server":
			cfg.Server = value
		case "token-file":
			cfg.TokenFile = value
		default:
			return fmt.Errorf("unknown config key %q (valid: server, token-file)", key)
		}
		if err := writeCliConfig(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "set %s = %s\n", key, value)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
}
