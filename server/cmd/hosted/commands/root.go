package commands

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hosted",
	Short: "Self-hosted application CLI",
	Long:  "Install and manage the Leamout self-hosted application.",
}

func Execute() error {
	return rootCmd.Execute()
}
