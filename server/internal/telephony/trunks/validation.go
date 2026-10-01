package trunks

import (
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

var (
	directions = map[string]struct{}{
		"inbound":       {},
		"outbound":      {},
		"bidirectional": {},
	}
	statuses = map[string]struct{}{
		"active":   {},
		"disabled": {},
	}
	transports = map[string]struct{}{
		"udp": {},
		"tcp": {},
		"tls": {},
	}
	supportedCodecs = map[string]struct{}{
		"PCMU": {},
		"PCMA": {},
		"G722": {},
		"OPUS": {},
		"G729": {},
	}
)

func normalizeCreate(req *CreateRequest) error {
	name, err := normalizeName(req.Name)
	if err != nil {
		return err
	}
	req.Name = name

	if req.Direction != nil {
		value, err := normalizeChoice(*req.Direction, directions, "direction")
		if err != nil {
			return err
		}
		req.Direction = &value
	}
	if req.Status != nil {
		value, err := normalizeChoice(*req.Status, statuses, "status")
		if err != nil {
			return err
		}
		req.Status = &value
	}
	if req.MaxCPS != nil && *req.MaxCPS < 1 {
		return apperror.NewBadRequest("max_cps must be greater than zero")
	}
	if req.MaxConcurrentCalls != nil && *req.MaxConcurrentCalls < 1 {
		return apperror.NewBadRequest("max_concurrent_calls must be greater than zero")
	}

	codecs, err := normalizeCodecs(req.Codecs)
	if err != nil {
		return err
	}
	req.Codecs = codecs

	if req.OutboundCredential != nil {
		if err := normalizeCredential(req.OutboundCredential); err != nil {
			return apperror.NewBadRequest("outbound credential: " + err.Error())
		}
	}

	method := "ip"
	if req.InboundAuthMethod != nil {
		method = strings.ToLower(strings.TrimSpace(*req.InboundAuthMethod))
	}
	if method != "ip" && method != "digest" && method != "none" {
		return apperror.NewBadRequest("inbound_auth_method must be ip, digest, or none")
	}
	req.InboundAuthMethod = &method

	if method == "digest" {
		if req.InboundCredential == nil {
			return apperror.NewBadRequest(
				"inbound credential is required for digest authentication",
			)
		}
		if err := normalizeCredential(req.InboundCredential); err != nil {
			return apperror.NewBadRequest("inbound credential: " + err.Error())
		}
	} else if req.InboundCredential != nil {
		return apperror.NewBadRequest(
			"inbound credential is only accepted for digest authentication",
		)
	}

	return nil
}

func normalizeUpdate(req *UpdateRequest) error {
	if req.Name == nil &&
		req.Direction == nil &&
		req.Status == nil &&
		req.InboundEnabled == nil &&
		req.MaxCPS == nil &&
		req.MaxConcurrentCalls == nil &&
		req.Codecs == nil &&
		req.SupportsVideo == nil &&
		req.SupportsFax == nil {
		return apperror.NewBadRequest("at least one field is required")
	}

	if req.Name != nil {
		value, err := normalizeName(*req.Name)
		if err != nil {
			return err
		}
		req.Name = &value
	}
	if req.Direction != nil {
		value, err := normalizeChoice(*req.Direction, directions, "direction")
		if err != nil {
			return err
		}
		req.Direction = &value
	}
	if req.Status != nil {
		value, err := normalizeChoice(*req.Status, statuses, "status")
		if err != nil {
			return err
		}
		req.Status = &value
	}
	if req.MaxCPS != nil && *req.MaxCPS < 1 {
		return apperror.NewBadRequest("max_cps must be greater than zero")
	}
	if req.MaxConcurrentCalls != nil && *req.MaxConcurrentCalls < 1 {
		return apperror.NewBadRequest("max_concurrent_calls must be greater than zero")
	}
	if req.Codecs != nil {
		value, err := normalizeCodecs(*req.Codecs)
		if err != nil {
			return err
		}
		req.Codecs = &value
	}

	return nil
}

func normalizeAuth(req *AuthRequest, inbound bool) error {
	req.Method = strings.ToLower(strings.TrimSpace(req.Method))
	allowed := req.Method == "digest" ||
		(!inbound && req.Method == "none") ||
		(inbound && (req.Method == "ip" || req.Method == "none"))
	if !allowed {
		return apperror.NewBadRequest("invalid authentication method")
	}

	if req.Method != "digest" {
		if req.Username != nil || req.Realm != nil || req.Secret != nil {
			return apperror.NewBadRequest(
				"username, realm, and secret are only accepted for digest authentication",
			)
		}
		return nil
	}

	if req.Username == nil || req.Realm == nil || req.Secret == nil {
		return apperror.NewBadRequest(
			"username, realm, and secret are required for digest authentication",
		)
	}

	credential := &DigestCredential{
		Username: *req.Username,
		Realm:    *req.Realm,
		Secret:   *req.Secret,
	}
	if err := normalizeCredential(credential); err != nil {
		return apperror.NewBadRequest(err.Error())
	}
	req.Username = &credential.Username
	req.Realm = &credential.Realm
	req.Secret = &credential.Secret
	return nil
}

func normalizeName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 255 {
		return "", apperror.NewBadRequest(
			"name must be between 1 and 255 characters",
		)
	}
	return value, nil
}

func normalizeCredential(credential *DigestCredential) error {
	credential.Username = strings.TrimSpace(credential.Username)
	credential.Realm = strings.TrimSpace(credential.Realm)

	if credential.Username == "" || len(credential.Username) > 255 {
		return fmt.Errorf("username must be between 1 and 255 characters")
	}
	if credential.Secret == "" || len(credential.Secret) > 4096 {
		return fmt.Errorf("secret must be between 1 and 4096 characters")
	}
	if credential.Realm == "" ||
		len(credential.Realm) > 255 ||
		strings.ContainsAny(credential.Realm, "\r\n") {
		return fmt.Errorf("realm must be between 1 and 255 characters")
	}
	return nil
}

func normalizeCodecs(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if _, ok := supportedCodecs[value]; !ok {
			return nil, apperror.NewBadRequest(
				fmt.Sprintf("unsupported codec %q", value),
			)
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
		return netip.Prefix{}, apperror.NewBadRequest(
			"cidr must be a valid network prefix",
		)
	}
	return prefix.Masked(), nil
}

func validateID(id uuid.UUID, field string) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest(field + " is required")
	}
	return nil
}

func normalizeChoice(
	value string,
	choices map[string]struct{},
	field string,
) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if _, ok := choices[value]; !ok {
		return "", apperror.NewBadRequest("invalid " + field)
	}
	return value, nil
}

func normalizeHost(value string) (string, error) {
	value = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(value), "."))
	if value == "" ||
		len(value) > 253 ||
		strings.ContainsAny(value, " \t\r\n") {
		return "", apperror.NewBadRequest(
			"host must be a valid IP address or hostname",
		)
	}
	if net.ParseIP(value) != nil {
		return value, nil
	}

	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 ||
			len(label) > 63 ||
			label[0] == '-' ||
			label[len(label)-1] == '-' {
			return "", apperror.NewBadRequest(
				"host must be a valid IP address or hostname",
			)
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') &&
				(char < '0' || char > '9') &&
				char != '-' {
				return "", apperror.NewBadRequest(
					"host must be a valid IP address or hostname",
				)
			}
		}
	}
	return value, nil
}

func validatePort(port int32) error {
	if port < 1 || port > 65535 {
		return apperror.NewBadRequest("port must be between 1 and 65535")
	}
	return nil
}

func validatePriority(priority int32) error {
	if priority < 0 {
		return apperror.NewBadRequest("priority must be non-negative")
	}
	return nil
}

func validateWeight(weight int32) error {
	if weight < 1 {
		return apperror.NewBadRequest("weight must be greater than zero")
	}
	return nil
}
