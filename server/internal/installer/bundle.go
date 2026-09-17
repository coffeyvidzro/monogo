package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const bundleComposePath = "deploy/docker/compose.yaml"

var requiredBundleFiles = []string{
	bundleComposePath,
	"deploy/docker/Caddyfile",
	"containers/nats/nats-server.conf",
	"containers/coturn/turnserver.conf",
	"server/scripts/certs/certbot-hook.sh",
	"server/scripts/certs/setup-certbot.sh",
	"server/scripts/certs/issue-certificates.sh",
	"server/scripts/deploy/lib.sh",
	"server/scripts/deploy/preflight.sh",
	"server/scripts/deploy/up.sh",
	"server/scripts/deploy/verify.sh",
	"server/scripts/deploy/restart.sh",
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
	for _, path := range requiredBundleFiles {
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

	if err := copyDir(filepath.Join(bundleDir, "server/scripts/certs"), filepath.Join(releaseDir, "scripts/certs"), 0o755); err != nil {
		return fmt.Errorf("install certificate scripts: %w", err)
	}
	if err := copyDir(filepath.Join(bundleDir, "server/scripts/deploy"), filepath.Join(releaseDir, "scripts/deploy"), 0o755); err != nil {
		return fmt.Errorf("install deploy scripts: %w", err)
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
