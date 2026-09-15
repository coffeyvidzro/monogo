package commands

import (
	"fmt"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/coffeyvidzro/monogo/internal/installer"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Leamout",
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("🚀 Initializing Leamout...")
		fmt.Println()

		config, err := promptInstallConfig()
		if err != nil {
			return err
		}

		fmt.Println()
		color.Yellow("Installation configuration:")
		fmt.Printf("  Domain:       %s\n", config.Domain)
		fmt.Printf("  Public IPv4:  %s\n", config.PublicIP)
		fmt.Printf("  Version:      %s\n", config.Version)
		fmt.Printf("  Runtime:      %s\n", installer.InstallRoot)
		fmt.Printf("  Configuration:%s\n", installer.ConfigRoot)
		fmt.Printf("  State:        %s\n", installer.StateRoot)
		fmt.Println("  TLS:          Let's Encrypt (automatic for SIP/TURN)")
		fmt.Println()

		var confirmed bool
		if err := survey.AskOne(&survey.Confirm{
			Message: "Proceed with installation?",
			Default: true,
		}, &confirmed); err != nil {
			return err
		}
		if !confirmed {
			color.Yellow("Installation cancelled.")
			return nil
		}

		fmt.Println()
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " Installing Leamout..."
		s.Start()

		err = installer.Install(config)
		s.Stop()
		if err != nil {
			color.Red("✖ Installation failed")
			return err
		}

		color.Green("✔ Installation completed successfully!")
		fmt.Printf("Release:       %s\n", installer.ReleaseDir(config.Version))
		fmt.Printf("Configuration: %s\n", installer.ConfigRoot)
		return nil
	},
}

func promptInstallConfig() (*installer.Config, error) {
	config := &installer.Config{
		Version: installer.DefaultVersion(),
	}
	prompts := []*survey.Question{
		{
			Name: "domain",
			Prompt: &survey.Input{
				Message: "Base domain (for api., sip., and turn.):",
			},
		},
		{
			Name: "publicIP",
			Prompt: &survey.Input{
				Message: "Public IPv4 address:",
			},
		},
	}

	if err := survey.Ask(prompts, config); err != nil {
		return nil, err
	}
	return config, nil
}

func init() {
	rootCmd.AddCommand(installCmd)
}
