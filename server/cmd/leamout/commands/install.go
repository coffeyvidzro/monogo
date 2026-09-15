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
	Short: "Install the self-hosted application",
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("🚀 Initializing Leamout Self-Hosted...")
		fmt.Println()

		config, err := promptInstallConfig()
		if err != nil {
			return err
		}

		fmt.Println()

		color.Yellow("Installation configuration:")
		fmt.Printf("  Domain:           %s\n", config.Domain)
		fmt.Printf("  PostgreSQL:       %s:%s\n", config.PostgresHost, config.PostgresPort)
		fmt.Printf("  Database:         %s\n", config.PostgresDatabase)
		fmt.Printf("  FreeSWITCH:       %t\n", config.InstallFreeSWITCH)
		fmt.Printf("  Redis:            %t\n", config.InstallRedis)
		fmt.Printf("  NATS:             %t\n", config.InstallNATS)

		fmt.Println()

		var confirmed bool

		confirmPrompt := &survey.Confirm{
			Message: "Proceed with installation?",
			Default: true,
		}

		if err := survey.AskOne(confirmPrompt, &confirmed); err != nil {
			return err
		}

		if !confirmed {
			color.Yellow("Installation cancelled.")
			return nil
		}

		fmt.Println()

		s := spinner.New(
			spinner.CharSets[14],
			100*time.Millisecond,
		)

		s.Suffix = " Installing Leamout Self-Hosted..."
		s.Start()

		err = installer.Install(config)

		s.Stop()

		if err != nil {
			color.Red("✖ Installation failed")
			return err
		}

		color.Green("✔ Installation completed successfully!")

		return nil
	},
}

func promptInstallConfig() (*installer.Config, error) {
	config := &installer.Config{}

	prompts := []*survey.Question{
		{
			Name: "domain",
			Prompt: &survey.Input{
				Message: "Domain:",
				Default: "localhost",
			},
		},
		{
			Name: "postgresHost",
			Prompt: &survey.Input{
				Message: "PostgreSQL host:",
				Default: "localhost",
			},
		},
		{
			Name: "postgresPort",
			Prompt: &survey.Input{
				Message: "PostgreSQL port:",
				Default: "5432",
			},
		},
		{
			Name: "postgresDatabase",
			Prompt: &survey.Input{
				Message: "PostgreSQL database:",
				Default: "leamout",
			},
		},
		{
			Name: "postgresUser",
			Prompt: &survey.Input{
				Message: "PostgreSQL username:",
				Default: "postgres",
			},
		},
		{
			Name: "postgresPassword",
			Prompt: &survey.Password{
				Message: "PostgreSQL password:",
			},
		},
		{
			Name: "installFreeSWITCH",
			Prompt: &survey.Confirm{
				Message: "Install FreeSWITCH?",
				Default: true,
			},
		},
		{
			Name: "installRedis",
			Prompt: &survey.Confirm{
				Message: "Install Redis?",
				Default: true,
			},
		},
		{
			Name: "installNATS",
			Prompt: &survey.Confirm{
				Message: "Install NATS?",
				Default: true,
			},
		},
	}

	err := survey.Ask(prompts, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func init() {
	rootCmd.AddCommand(installCmd)
}
