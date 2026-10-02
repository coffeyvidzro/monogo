package storage

import (
	"net"
	"net/url"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
)

func normalizeCreate(req *CreateRequest) error {
	return normalizeCreateWithPolicy(req, false)
}

func normalizeCreateWithPolicy(req *CreateRequest, allowPrivateEndpoints bool) error {
	req.Name = strings.TrimSpace(req.Name)
	req.EndpointURL = strings.TrimSpace(req.EndpointURL)
	req.Region = strings.TrimSpace(req.Region)
	req.Bucket = strings.TrimSpace(req.Bucket)
	req.AccessKeyID = strings.TrimSpace(req.AccessKeyID)
	req.SecretAccessKey = strings.TrimSpace(req.SecretAccessKey)

	if req.Region == "" {
		req.Region = "us-east-1"
	}
	if err := validateName(req.Name); err != nil {
		return err
	}
	if err := validateEndpoint(req.EndpointURL, allowPrivateEndpoints); err != nil {
		return err
	}
	if err := validateRegion(req.Region); err != nil {
		return err
	}
	if err := validateBucket(req.Bucket); err != nil {
		return err
	}
	if req.AccessKeyID == "" {
		return apperror.NewBadRequest("access_key_id is required")
	}
	if req.SecretAccessKey == "" {
		return apperror.NewBadRequest("secret_access_key is required")
	}
	return nil
}

func normalizeUpdate(req *UpdateRequest) error {
	if req.Name == nil && req.AccessKeyID == nil && req.SecretAccessKey == nil && req.Status == nil {
		return apperror.NewBadRequest("at least one field is required")
	}
	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		if err := validateName(value); err != nil {
			return err
		}
		req.Name = &value
	}
	if req.AccessKeyID != nil {
		value := strings.TrimSpace(*req.AccessKeyID)
		if value == "" {
			return apperror.NewBadRequest("access_key_id cannot be empty")
		}
		req.AccessKeyID = &value
	}
	if req.SecretAccessKey != nil {
		value := strings.TrimSpace(*req.SecretAccessKey)
		if value == "" {
			return apperror.NewBadRequest("secret_access_key cannot be empty")
		}
		req.SecretAccessKey = &value
	}
	if req.Status != nil {
		value := strings.ToLower(strings.TrimSpace(*req.Status))
		if value != StatusActive && value != StatusDisabled {
			return apperror.NewBadRequest("status must be active or disabled")
		}
		req.Status = &value
	}
	return nil
}

func validateName(value string) error {
	if len(value) < 1 || len(value) > 128 {
		return apperror.NewBadRequest("name must be between 1 and 128 characters")
	}
	return nil
}

func validateEndpoint(value string, allowPrivateEndpoints bool) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") {
		return apperror.NewBadRequest("endpoint_url must be an HTTP or HTTPS origin")
	}
	if !allowPrivateEndpoints && parsed.Scheme != "https" {
		return apperror.NewBadRequest("endpoint_url must use HTTPS")
	}

	hostname := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if allowPrivateEndpoints {
		if ip := net.ParseIP(hostname); ip != nil && unsafeSelfHostedIP(ip) {
			return apperror.NewBadRequest("endpoint_url uses a disallowed network host")
		}
		return nil
	}

	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") {
		return apperror.NewBadRequest("endpoint_url must use a public network host")
	}
	if ip := net.ParseIP(hostname); ip != nil && !publicIP(ip) {
		return apperror.NewBadRequest("endpoint_url must use a public network host")
	}
	return nil
}

func publicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsUnspecified()
}

func unsafeSelfHostedIP(ip net.IP) bool {
	return ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

func validateRegion(value string) error {
	if len(value) < 1 || len(value) > 128 {
		return apperror.NewBadRequest("region must be between 1 and 128 characters")
	}
	return nil
}

func validateBucket(value string) error {
	if len(value) < 1 || len(value) > 255 {
		return apperror.NewBadRequest("bucket must be between 1 and 255 characters")
	}
	return nil
}
