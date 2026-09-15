package installer

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

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
	if err := writeSecretFile(filepath.Join(InstallStateDir, "installation.json"), content); err != nil {
		return fmt.Errorf("write installation state: %w", err)
	}
	return nil
}
