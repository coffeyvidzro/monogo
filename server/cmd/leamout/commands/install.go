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
		fmt.Printf("  Public IP:    %s\n", config.PublicIP)
		fmt.Printf("  Version:      %s\n", config.Version)
		fmt.Printf("  Install dir:  %s\n", config.InstallDir)
		fmt.Printf("  TLS cert:     %s\n", config.TLSCertificate)
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
		fmt.Printf("Deployment directory: %s\n", config.InstallDir)
		return nil
	},
}

func promptInstallConfig() (*installer.Config, error) {
	config := &installer.Config{}
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
				Message: "Public IP address:",
			},
		},
		{
			Name: "version",
			Prompt: &survey.Input{
				Message: "Leamout version:",
				Default: installer.DefaultVersion(),
			},
		},
		{
			Name: "installDir",
			Prompt: &survey.Input{
				Message: "Install directory:",
				Default: "/opt/leamout",
			},
		},
		{
			Name: "tlsCertificate",
			Prompt: &survey.Input{
				Message: "TLS certificate for sip/turn (fullchain.pem):",
			},
		},
		{
			Name: "tlsPrivateKey",
			Prompt: &survey.Input{
				Message: "TLS private key path for sip/turn:",
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
