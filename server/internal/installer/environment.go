package installer

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

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
