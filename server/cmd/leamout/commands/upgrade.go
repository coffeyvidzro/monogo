package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade a Leamout installation",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("upgrade is not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
