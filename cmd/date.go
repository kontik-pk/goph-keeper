package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var buildDate = ""

// dateCmd represents the date command
var dateCmd = &cobra.Command{
	Use:     "build-date",
	Short:   "Show build date",
	Example: "goph-keeper build-date",
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("build date: %s\n", buildDate)
	},
}

func init() {
	rootCmd.AddCommand(dateCmd)
}
