package installer

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const bundleComposePath = "deploy/docker/compose.yaml"

const certbotDeployHookPath = "/etc/letsencrypt/renewal-hooks/deploy/leamout"

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

	if err := ensureCertbot(); err != nil {
		return err
	}
	if err := installCertificateDeployHook(); err != nil {
		return err
	}
	if err := issueCertificates(config); err != nil {
		return err
	}

	if err := writeSecretFileExclusive(EnvironmentPath, environment); err != nil {
		return fmt.Errorf("write deployment environment: %w", err)
	}
	if err := runCompose(EnvironmentPath, "pull"); err != nil {
		return fmt.Errorf("pull Leamout containers: %w", err)
	}
	if err := runCompose(EnvironmentPath, "up", "-d"); err != nil {
		return fmt.Errorf("start Leamout: %w", err)
	}
	if err := writeInstallationState(config); err != nil {
		return err
	}

	return nil
}

func validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("installation configuration is required")
	}

	config.Domain = strings.TrimSpace(config.Domain)
	config.PublicIP = strings.TrimSpace(config.PublicIP)
	config.Version = strings.TrimSpace(config.Version)

	if config.Domain == "" || strings.Contains(config.Domain, "://") || strings.ContainsAny(config.Domain, " /#") {
		return fmt.Errorf("a valid base domain is required")
	}
	ip := net.ParseIP(config.PublicIP)
	if ip == nil || ip.To4() == nil {
		return fmt.Errorf("a valid public IPv4 address is required")
	}
	if config.Version == "" || strings.ContainsAny(config.Version, `/\\`) {
		return fmt.Errorf("Leamout version is required")
	}
	return nil
}

func validateDNS(config *Config) error {
	for _, host := range []string{"api." + config.Domain, "sip." + config.Domain, "turn." + config.Domain} {
		addresses, err := net.LookupIP(host)
		if err != nil {
			return fmt.Errorf("DNS for %s must resolve to %s before installation: %w", host, config.PublicIP, err)
		}

		matchedIPv4 := false
		for _, address := range addresses {
			if address.To4() == nil {
				return fmt.Errorf("DNS for %s has an AAAA record but this Self-Hosted release is IPv4-only; remove the AAAA record before installation", host)
			}
			if address.String() == config.PublicIP {
				matchedIPv4 = true
			}
		}
		if !matchedIPv4 {
			return fmt.Errorf("DNS for %s does not resolve to %s", host, config.PublicIP)
		}
	}
	return nil
}

func validateDocker() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("Docker is required: %w", err)
	}
	cmd := exec.Command("docker", "compose", "version")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Docker Compose plugin is required: %s: %w", strings.TrimSpace(string(output)), err)
	}
	cmd = exec.Command("docker", "info")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Docker daemon is unavailable: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func prepareFilesystem() error {
	for _, directory := range []struct {
		path string
		mode os.FileMode
	}{
		{InstallRoot, 0o755},
		{ReleasesDir, 0o755},
		{ConfigRoot, 0o700},
		{CertificateDir, 0o700},
		{LicenseDir, 0o700},
		{StateRoot, 0o700},
		{InstallStateDir, 0o700},
		{ACMEWebroot, 0o755},
		{BackupsDir, 0o700},
		{StagingDir, 0o700},
		{LogRoot, 0o700},
		{RunRoot, 0o700},
	} {
		if err := ensureDirectory(directory.path, directory.mode); err != nil {
			return err
		}
	}
	return nil
}

func ensureDirectory(path string, mode os.FileMode) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, mode); err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}
		return os.Chmod(path, mode)
	}
	if err != nil {
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("refusing unsafe managed path %s", path)
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("secure %s: %w", path, err)
	}
	return nil
}

func findBundleDir() (string, error) {
	candidates := make([]string, 0, 3)
	if override := strings.TrimSpace(os.Getenv("LEAMOUT_BUNDLE_DIR")); override != "" {
		candidates = append(candidates, override)
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(executable))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}

	for _, candidate := range candidates {
		if bundleComplete(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("self-hosted release bundle not found; run leamout from the extracted release bundle")
}

func bundleComplete(root string) bool {
	required := []string{
		bundleComposePath,
		"deploy/docker/Caddyfile",
		"server/migrations/atlas.sum",
		"containers/nats/nats-server.conf",
		"containers/coturn/turnserver.conf",
	}
	for _, path := range required {
		if info, err := os.Stat(filepath.Join(root, path)); err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

func installBundleFiles(bundleDir, releaseDir, version string) error {
	if err := ensureDirectory(releaseDir, 0o755); err != nil {
		return err
	}
	files := map[string]string{
		bundleComposePath:                    "compose.yaml",
		"deploy/docker/Caddyfile":           "Caddyfile",
		"containers/nats/nats-server.conf":  "config/nats-server.conf",
		"containers/coturn/turnserver.conf": "config/turnserver.conf",
	}
	for source, destination := range files {
		if err := copyFile(filepath.Join(bundleDir, source), filepath.Join(releaseDir, destination), 0o644); err != nil {
			return fmt.Errorf("install %s: %w", source, err)
		}
	}
	if err := copyDir(filepath.Join(bundleDir, "server/migrations"), filepath.Join(releaseDir, "migrations")); err != nil {
		return fmt.Errorf("install migrations: %w", err)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, "VERSION"), []byte(version+"\n"), 0o644); err != nil {
		return fmt.Errorf("write release version: %w", err)
	}
	return nil
}

func activateRelease(releaseDir string) error {
	info, err := os.Lstat(CurrentPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("refusing to replace non-symlink %s", CurrentPath)
		}
		if err := os.Remove(CurrentPath); err != nil {
			return fmt.Errorf("replace current release: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect current release: %w", err)
	}
	if err := os.Symlink(releaseDir, CurrentPath); err != nil {
		return fmt.Errorf("activate release: %w", err)
	}
	return nil
}

func buildEnvironment(config *Config) ([]byte, error) {
	deploymentID, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generate deployment ID: %w", err)
	}
	postgresPassword, err := randomSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generate PostgreSQL password: %w", err)
	}
	freeSWITCHPassword, err := randomSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generate FreeSWITCH password: %w", err)
	}
	credentialKey, err := randomSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generate carrier credential encryption key: %w", err)
	}
	operatorSecret, err := randomSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generate operator API secret: %w", err)
	}
	turnSecret, err := randomSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generate TURN auth secret: %w", err)
	}

	turnHost := "turn." + config.Domain
	content := strings.Join([]string{
		"DOMAIN=" + config.Domain,
		"PUBLIC_IP=" + config.PublicIP,
		"LEAMOUT_VERSION=" + config.Version,
		"LEAMOUT_DEPLOYMENT_ID=selfhost-" + deploymentID,
		"POSTGRES_PASSWORD=" + postgresPassword,
		"FREESWITCH_ESL_PASSWORD=" + freeSWITCHPassword,
		"CARRIER_CREDENTIAL_ENCRYPTION_KEY=" + credentialKey,
		"OPERATOR_API_SECRET=" + operatorSecret,
		"TURN_AUTH_SECRET=" + turnSecret,
		"TURN_REALM=" + turnHost,
		"TURN_PUBLIC_URLS=stun:" + turnHost + ":3478,turn:" + turnHost + ":3478?transport=udp,turn:" + turnHost + ":3478?transport=tcp,turns:" + turnHost + ":5349?transport=tcp",
		"CORS_ORIGINS=https://" + config.Domain + ",https://api." + config.Domain,
		"",
	}, "\n")
	return []byte(content), nil
}

func writeSecretFile(path string, content []byte) error {
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func writeSecretFileExclusive(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func ensureCertbot() error {
	if _, err := exec.LookPath("apt-get"); err != nil {
		return fmt.Errorf("automatic Let's Encrypt setup currently requires an apt-based Linux host")
	}
	if _, err := exec.LookPath("certbot"); err != nil {
		update := exec.Command("apt-get", "update")
		update.Stdout = os.Stdout
		update.Stderr = os.Stderr
		if err := update.Run(); err != nil {
			return fmt.Errorf("update package index for Certbot: %w", err)
		}
		install := exec.Command("apt-get", "install", "-y", "certbot")
		install.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		install.Stdout = os.Stdout
		install.Stderr = os.Stderr
		if err := install.Run(); err != nil {
			return fmt.Errorf("install Certbot: %w", err)
		}
	}

	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemd is required for automatic Certbot renewal")
	}
	timer := exec.Command("systemctl", "enable", "--now", "certbot.timer")
	timer.Stdout = os.Stdout
	timer.Stderr = os.Stderr
	if err := timer.Run(); err != nil {
		return fmt.Errorf("enable automatic Certbot renewal: %w", err)
	}
	return nil
}

func installCertificateDeployHook() error {
	hookDir := filepath.Dir(certbotDeployHookPath)
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		return fmt.Errorf("create Certbot deploy hook directory: %w", err)
	}
	hook := `#!/bin/sh
set -eu

umask 077

[ -n "${RENEWED_LINEAGE:-}" ] || exit 0

install -d -o root -g root -m 0700 /etc/leamout/certs
install -o root -g root -m 0644 "$RENEWED_LINEAGE/fullchain.pem" /etc/leamout/certs/fullchain.pem.new
install -o root -g 65534 -m 0640 "$RENEWED_LINEAGE/privkey.pem" /etc/leamout/certs/privkey.pem.new
mv /etc/leamout/certs/fullchain.pem.new /etc/leamout/certs/fullchain.pem
mv /etc/leamout/certs/privkey.pem.new /etc/leamout/certs/privkey.pem

if [ -f /etc/leamout/leamout.env ] && [ -L /opt/leamout/current ]; then
  services=""
  for service in opensips coturn; do
    if docker compose --env-file /etc/leamout/leamout.env -f /opt/leamout/current/compose.yaml ps -q "$service" 2>/dev/null | grep -q .; then
      services="$services $service"
    fi
  done
  [ -z "$services" ] || docker compose --env-file /etc/leamout/leamout.env -f /opt/leamout/current/compose.yaml restart $services
fi
`
	if err := os.WriteFile(certbotDeployHookPath, []byte(hook), 0o700); err != nil {
		return fmt.Errorf("write Certbot deploy hook: %w", err)
	}
	return os.Chmod(certbotDeployHookPath, 0o700)
}

func issueCertificates(config *Config) error {
	cmd := exec.Command(
		"certbot", "certonly",
		"--non-interactive",
		"--agree-tos",
		"--register-unsafely-without-email",
		"--cert-name", "leamout-sip-turn",
		"--webroot",
		"--webroot-path", ACMEWebroot,
		"-d", "sip."+config.Domain,
		"-d", "turn."+config.Domain,
		"--deploy-hook", certbotDeployHookPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("obtain SIP/TURN certificate from Let's Encrypt: %w", err)
	}
	for _, path := range []string{filepath.Join(CertificateDir, "fullchain.pem"), filepath.Join(CertificateDir, "privkey.pem")} {
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("Certbot did not deploy required certificate material to %s", CertificateDir)
		}
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
	if err := writeSecretFile(filepath.Join(InstallStateDir, "installation.json"), content); err != nil {
		return fmt.Errorf("write installation state: %w", err)
	}
	return nil
}

func randomSecret(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func copyDir(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported bundle file type: %s", path)
		}
		return copyFile(path, target, 0o644)
	})
}

func copyFile(source, destination string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(destination, mode)
}

func runCompose(environmentPath string, args ...string) error {
	composeArgs := []string{
		"compose",
		"--env-file", environmentPath,
		"-f", filepath.Join(CurrentPath, "compose.yaml"),
	}
	composeArgs = append(composeArgs, args...)
	cmd := exec.Command("docker", composeArgs...)
	cmd.Dir = CurrentPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
