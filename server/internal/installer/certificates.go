package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func installCertificateLifecycle(config *Config) error {
	certDir := filepath.Join(CurrentPath, "scripts", "certs")
	installRenewal := filepath.Join(certDir, "install-certbot-renewal.sh")
	provision := filepath.Join(certDir, "provision-letsencrypt.sh")

	if err := runHostScript(installRenewal); err != nil {
		return fmt.Errorf("install Certbot renewal lifecycle: %w", err)
	}
	if err := runHostScript(provision, config.Domain, ACMEWebroot); err != nil {
		return fmt.Errorf("obtain SIP/TURN certificate from Let's Encrypt: %w", err)
	}

	for _, path := range []string{
		filepath.Join(CertificateDir, "fullchain.pem"),
		filepath.Join(CertificateDir, "privkey.pem"),
	} {
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("Certbot did not deploy required certificate material to %s", CertificateDir)
		}
	}
	return nil
}

func runHostScript(path string, args ...string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect host helper %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("host helper is not a regular file: %s", path)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("host helper is not executable: %s", path)
	}

	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
