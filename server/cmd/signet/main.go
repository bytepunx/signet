package main

import "os"

// version is set at build time via -ldflags="-X main.version=<tag>".
var version = "dev"

func main() {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
