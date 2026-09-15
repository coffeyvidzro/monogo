package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Leamout installation status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("status is not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
