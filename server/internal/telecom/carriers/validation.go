package carriers

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/google/uuid"
)

var supportedCodecs = map[string]struct{}{"PCMU": {}, "PCMA": {}, "G722": {}, "OPUS": {}, "G729": {}}

func normalizeCreate(req *CreateRequest) error {
	if req.ProviderID == uuid.Nil {
		return fmt.Errorf("provider_id is required")
	}
	name, err := normalizeName(req.Name)
	if err != nil {
		return err
	}
	req.Name = name
	if req.Status != nil {
		v := strings.ToLower(strings.TrimSpace(*req.Status))
		if v != "active" && v != "disabled" {
			return fmt.Errorf("status must be active or disabled")
		}
		req.Status = &v
	}
	if req.MaxCPS != nil && *req.MaxCPS < 1 {
		return fmt.Errorf("max_cps must be greater than zero")
	}
	if req.MaxConcurrentCalls != nil && *req.MaxConcurrentCalls < 1 {
		return fmt.Errorf("max_concurrent_calls must be greater than zero")
	}
	if req.MaxDailyMinutes != nil && *req.MaxDailyMinutes < 1 {
		return fmt.Errorf("max_daily_minutes must be greater than zero")
	}
	codecs, err := normalizeCodecs(req.Codecs)
	if err != nil {
		return err
	}
	req.Codecs = codecs
	if req.OutboundCredential != nil {
		if err := normalizeCredential(req.OutboundCredential); err != nil {
			return fmt.Errorf("outbound credential: %w", err)
		}
	}
	method := "ip"
	if req.InboundAuthMethod != nil {
		method = strings.ToLower(strings.TrimSpace(*req.InboundAuthMethod))
	}
	if method != "ip" && method != "digest" && method != "none" {
		return fmt.Errorf("inbound_auth_method must be ip, digest, or none")
	}
	req.InboundAuthMethod = &method
	if method == "digest" {
		if req.InboundCredential == nil {
			return fmt.Errorf("inbound credential is required for digest authentication")
		}
		if err := normalizeCredential(req.InboundCredential); err != nil {
			return fmt.Errorf("inbound credential: %w", err)
		}
	} else if req.InboundCredential != nil {
		return fmt.Errorf("inbound credential is only accepted for digest authentication")
	}
	return nil
}

func normalizeUpdate(req *UpdateRequest) error {
	if req.Name == nil && req.Status == nil && req.InboundEnabled == nil && req.MaxCPS == nil && req.MaxConcurrentCalls == nil && req.MaxDailyMinutes == nil && req.Codecs == nil && req.SupportsVideo == nil && req.SupportsFax == nil {
		return fmt.Errorf("at least one field is required")
	}
	if req.Name != nil {
		v, err := normalizeName(*req.Name)
		if err != nil {
			return err
		}
		req.Name = &v
	}
	if req.Status != nil {
		v := strings.ToLower(strings.TrimSpace(*req.Status))
		if v != "active" && v != "disabled" {
			return fmt.Errorf("status must be active or disabled")
		}
		req.Status = &v
	}
	if req.MaxCPS != nil && *req.MaxCPS < 1 {
		return fmt.Errorf("max_cps must be greater than zero")
	}
	if req.MaxConcurrentCalls != nil && *req.MaxConcurrentCalls < 1 {
		return fmt.Errorf("max_concurrent_calls must be greater than zero")
	}
	if req.MaxDailyMinutes != nil && *req.MaxDailyMinutes < 1 {
		return fmt.Errorf("max_daily_minutes must be greater than zero")
	}
	if req.Codecs != nil {
		v, err := normalizeCodecs(*req.Codecs)
		if err != nil {
			return err
		}
		req.Codecs = &v
	}
	return nil
}

func normalizeAuth(req *AuthRequest, inbound bool) error {
	req.Method = strings.ToLower(strings.TrimSpace(req.Method))
	allowed := req.Method == "digest" || (!inbound && req.Method == "none") || (inbound && (req.Method == "ip" || req.Method == "none"))
	if !allowed {
		return fmt.Errorf("invalid authentication method")
	}
	if req.Method != "digest" {
		if req.Username != nil || req.Secret != nil {
			return fmt.Errorf("username and secret are only accepted for digest authentication")
		}
		return nil
	}
	if req.Username == nil || req.Secret == nil {
		return fmt.Errorf("username and secret are required for digest authentication")
	}
	c := &DigestCredential{Username: *req.Username, Secret: *req.Secret}
	if err := normalizeCredential(c); err != nil {
		return err
	}
	req.Username, req.Secret = &c.Username, &c.Secret
	return nil
}

func normalizeName(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 255 {
		return "", fmt.Errorf("name must be between 1 and 255 characters")
	}
	return v, nil
}
func normalizeCredential(c *DigestCredential) error {
	c.Username = strings.TrimSpace(c.Username)
	if c.Username == "" || len(c.Username) > 255 {
		return fmt.Errorf("username must be between 1 and 255 characters")
	}
	if c.Secret == "" || len(c.Secret) > 4096 {
		return fmt.Errorf("secret must be between 1 and 4096 characters")
	}
	return nil
}
func normalizeCodecs(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if _, ok := supportedCodecs[value]; !ok {
			return nil, fmt.Errorf("unsupported codec %q", value)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}
func parseCIDR(value string) (netip.Prefix, error) {
	prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("cidr must be a valid network prefix")
	}
	return prefix.Masked(), nil
}
