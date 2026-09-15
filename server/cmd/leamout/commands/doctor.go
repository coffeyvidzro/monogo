package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check Leamout installation health",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("doctor is not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
