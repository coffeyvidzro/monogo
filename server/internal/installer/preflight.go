package installer

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

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
