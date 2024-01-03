package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "v0.0.0"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "goph-keeper",
	Version: version,
	Short:   "GophKeeper is a client-server system that allows the user to safely and securely store logins, passwords, binary data and other sensitive information.",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
