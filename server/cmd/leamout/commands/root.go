package commands

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "leamout",
	Short: "Leamout application CLI",
	Long:  "Install and manage the Leamout application.",
}

func Execute() error {
	return rootCmd.Execute()
}
