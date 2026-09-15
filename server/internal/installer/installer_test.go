package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cert := filepath.Join(dir, "fullchain.pem")
	key := filepath.Join(dir, "privkey.pem")
	if err := os.WriteFile(cert, []byte("cert"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Domain:         "example.com",
		PublicIP:       "203.0.113.10",
		Version:        "1.0.0",
		InstallDir:     filepath.Join(dir, "install"),
		TLSCertificate: cert,
		TLSPrivateKey:  key,
	}
	if err := validateConfig(config); err != nil {
		t.Fatalf("validateConfig() error = %v", err)
	}
}

func TestWriteEnvironment(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	config := &Config{
		Domain:     "example.com",
		PublicIP:   "203.0.113.10",
		Version:    "1.2.3",
		InstallDir: dir,
	}

	if err := writeEnvironment(config, path); err != nil {
		t.Fatalf("writeEnvironment() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, expected := range []string{
		"DOMAIN=example.com",
		"PUBLIC_IP=203.0.113.10",
		"LEAMOUT_VERSION=1.2.3",
		"TURN_REALM=turn.example.com",
		"CARRIER_CREDENTIAL_ENCRYPTION_KEY=",
		"OPERATOR_API_SECRET=",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("environment missing %q", expected)
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf(".env permissions = %o, want 600", got)
	}
}

func TestDefaultVersionFromBundle(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{
		composeRelativePath,
		"deploy/docker/Caddyfile",
		"server/migrations/atlas.sum",
		"containers/nats/nats-server.conf",
		"containers/coturn/turnserver.conf",
	} {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("2.3.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LEAMOUT_BUNDLE_DIR", dir)
	if got := DefaultVersion(); got != "2.3.4" {
		t.Fatalf("DefaultVersion() = %q, want %q", got, "2.3.4")
	}
}
