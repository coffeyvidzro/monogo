package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	config := &Config{
		Domain:   "example.com",
		PublicIP: "203.0.113.10",
		Version:  "1.0.0",
	}
	if err := validateConfig(config); err != nil {
		t.Fatalf("validateConfig() error = %v", err)
	}
}

func TestValidateConfigRejectsIPv6(t *testing.T) {
	t.Parallel()

	config := &Config{
		Domain:   "example.com",
		PublicIP: "2001:db8::1",
		Version:  "1.0.0",
	}
	if err := validateConfig(config); err == nil {
		t.Fatal("validateConfig() accepted IPv6, want IPv4-only validation")
	}
}

func TestBuildEnvironment(t *testing.T) {
	t.Parallel()

	config := &Config{
		Domain:   "example.com",
		PublicIP: "203.0.113.10",
		Version:  "1.2.3",
	}

	content, err := buildEnvironment(config)
	if err != nil {
		t.Fatalf("buildEnvironment() error = %v", err)
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
}

func TestReleaseDir(t *testing.T) {
	t.Parallel()

	if got, want := ReleaseDir("1.2.3"), "/opt/leamout/releases/1.2.3"; got != want {
		t.Fatalf("ReleaseDir() = %q, want %q", got, want)
	}
}

func TestDefaultVersionFromBundle(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{
		bundleComposePath,
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
