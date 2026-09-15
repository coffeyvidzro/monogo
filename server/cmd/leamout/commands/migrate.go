package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run Leamout database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("migrate is not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
