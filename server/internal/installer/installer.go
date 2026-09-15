package installer

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const composeRelativePath = "deploy/docker/compose.yaml"

func Install(config *Config) error {
	if err := validateConfig(config); err != nil {
		return err
	}
	if err := validateDocker(); err != nil {
		return err
	}

	bundleDir, err := findBundleDir()
	if err != nil {
		return err
	}

	envPath := filepath.Join(config.InstallDir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		return fmt.Errorf("Leamout is already configured at %s", config.InstallDir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check existing installation: %w", err)
	}

	if err := installBundleFiles(bundleDir, config.InstallDir); err != nil {
		return err
	}
	if err := installTLS(config); err != nil {
		return err
	}
	if err := writeEnvironment(config, envPath); err != nil {
		return err
	}

	if err := runCompose(config.InstallDir, "config", "--quiet"); err != nil {
		return fmt.Errorf("validate Docker Compose deployment: %w", err)
	}
	if err := runCompose(config.InstallDir, "pull"); err != nil {
		return fmt.Errorf("pull Leamout containers: %w", err)
	}
	if err := runCompose(config.InstallDir, "up", "-d"); err != nil {
		return fmt.Errorf("start Leamout: %w", err)
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
	config.InstallDir = strings.TrimSpace(config.InstallDir)
	config.TLSCertificate = strings.TrimSpace(config.TLSCertificate)
	config.TLSPrivateKey = strings.TrimSpace(config.TLSPrivateKey)

	if config.Domain == "" || strings.Contains(config.Domain, "://") || strings.ContainsAny(config.Domain, " /#") {
		return fmt.Errorf("a valid base domain is required")
	}
	if net.ParseIP(config.PublicIP) == nil {
		return fmt.Errorf("a valid public IP address is required")
	}
	if config.Version == "" {
		return fmt.Errorf("Leamout version is required")
	}
	if config.InstallDir == "" || !filepath.IsAbs(config.InstallDir) {
		return fmt.Errorf("install directory must be an absolute path")
	}
	if err := requireRegularFile(config.TLSCertificate, "TLS certificate"); err != nil {
		return err
	}
	if err := requireRegularFile(config.TLSPrivateKey, "TLS private key"); err != nil {
		return err
	}

	return nil
}

func requireRegularFile(path, label string) error {
	if path == "" {
		return fmt.Errorf("%s path is required", label)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", label, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s must be a regular file", label)
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
		composeRelativePath,
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

func installBundleFiles(bundleDir, installDir string) error {
	files := []string{
		composeRelativePath,
		"deploy/docker/Caddyfile",
		"containers/nats/nats-server.conf",
		"containers/coturn/turnserver.conf",
	}
	for _, path := range files {
		if err := copyFile(filepath.Join(bundleDir, path), filepath.Join(installDir, path), 0o644); err != nil {
			return fmt.Errorf("install %s: %w", path, err)
		}
	}
	if err := copyDir(filepath.Join(bundleDir, "server/migrations"), filepath.Join(installDir, "server/migrations")); err != nil {
		return fmt.Errorf("install migrations: %w", err)
	}
	return nil
}

func installTLS(config *Config) error {
	certDir := filepath.Join(config.InstallDir, "deploy/docker/certs")
	if err := os.MkdirAll(certDir, 0o700); err != nil {
		return fmt.Errorf("create TLS directory: %w", err)
	}
	if err := copyFile(config.TLSCertificate, filepath.Join(certDir, "fullchain.pem"), 0o644); err != nil {
		return fmt.Errorf("install TLS certificate: %w", err)
	}
	if err := copyFile(config.TLSPrivateKey, filepath.Join(certDir, "privkey.pem"), 0o600); err != nil {
		return fmt.Errorf("install TLS private key: %w", err)
	}
	return nil
}

func writeEnvironment(config *Config, path string) error {
	deploymentID, err := randomHex(16)
	if err != nil {
		return fmt.Errorf("generate deployment ID: %w", err)
	}
	postgresPassword, err := randomSecret(32)
	if err != nil {
		return fmt.Errorf("generate PostgreSQL password: %w", err)
	}
	freeSWITCHPassword, err := randomSecret(32)
	if err != nil {
		return fmt.Errorf("generate FreeSWITCH password: %w", err)
	}
	credentialKey, err := randomSecret(32)
	if err != nil {
		return fmt.Errorf("generate carrier credential encryption key: %w", err)
	}
	operatorSecret, err := randomSecret(32)
	if err != nil {
		return fmt.Errorf("generate operator API secret: %w", err)
	}
	turnSecret, err := randomSecret(32)
	if err != nil {
		return fmt.Errorf("generate TURN auth secret: %w", err)
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

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create installation directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write deployment environment: %w", err)
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

func runCompose(installDir string, args ...string) error {
	composeArgs := []string{
		"compose",
		"--env-file", filepath.Join(installDir, ".env"),
		"-f", filepath.Join(installDir, composeRelativePath),
	}
	composeArgs = append(composeArgs, args...)
	cmd := exec.Command("docker", composeArgs...)
	cmd.Dir = installDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
