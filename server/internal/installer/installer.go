package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func Install(config *Config) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("Leamout installation must run as root")
	}
	if err := validateConfig(config); err != nil {
		return err
	}
	if err := validateDocker(); err != nil {
		return err
	}
	if err := validateDNS(config); err != nil {
		return err
	}

	bundleDir, err := findBundleDir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(EnvironmentPath); err == nil {
		return fmt.Errorf("Leamout is already configured at %s", ConfigRoot)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check existing installation: %w", err)
	}

	if err := prepareFilesystem(); err != nil {
		return err
	}

	releaseDir := ReleaseDir(config.Version)
	if err := installBundleFiles(bundleDir, releaseDir, config.Version); err != nil {
		return err
	}
	if err := activateRelease(releaseDir); err != nil {
		return err
	}

	environment, err := buildEnvironment(config)
	if err != nil {
		return err
	}
	temporaryEnvironment := filepath.Join(RunRoot, "install.env")
	if err := writeSecretFile(temporaryEnvironment, environment); err != nil {
		return fmt.Errorf("write temporary deployment environment: %w", err)
	}
	defer os.Remove(temporaryEnvironment)

	if err := runCompose(temporaryEnvironment, "config", "--quiet"); err != nil {
		return fmt.Errorf("validate Docker Compose deployment: %w", err)
	}
	if err := runCompose(temporaryEnvironment, "pull", "caddy"); err != nil {
		return fmt.Errorf("pull Caddy: %w", err)
	}
	if err := runCompose(temporaryEnvironment, "up", "-d", "caddy"); err != nil {
		return fmt.Errorf("start ACME web endpoint: %w", err)
	}

	if err := installCertificateLifecycle(config); err != nil {
		return err
	}
	if err := writeSecretFileExclusive(EnvironmentPath, environment); err != nil {
		return fmt.Errorf("write deployment environment: %w", err)
	}

	deployDir := filepath.Join(CurrentPath, "scripts", "deploy")
	if err := runHostScript(filepath.Join(deployDir, "up.sh")); err != nil {
		return fmt.Errorf("start Leamout: %w", err)
	}
	if err := runHostScript(filepath.Join(deployDir, "verify.sh")); err != nil {
		return fmt.Errorf("verify Leamout deployment: %w", err)
	}
	if err := writeInstallationState(config); err != nil {
		return err
	}

	return nil
}

func writeInstallationState(config *Config) error {
	state := struct {
		Version  string `json:"version"`
		Domain   string `json:"domain"`
		PublicIP string `json:"public_ip"`
	}{
		Version:  config.Version,
		Domain:   config.Domain,
		PublicIP: config.PublicIP,
	}

	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode installation state: %w", err)
	}
	content = append(content, '\n')

	statePath := filepath.Join(InstallStateDir, "installation.json")
	if err := writeSecretFile(statePath, content); err != nil {
		return fmt.Errorf("write installation state: %w", err)
	}
	return nil
}
