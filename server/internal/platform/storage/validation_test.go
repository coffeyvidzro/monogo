package storage

import "testing"

func TestNormalizeCreateDefaultsRegion(t *testing.T) {
	req := CreateRequest{
		Name:            " Recording archive ",
		EndpointURL:     "https://s3.example.com",
		Bucket:          " recordings ",
		AccessKeyID:     " access ",
		SecretAccessKey: " secret ",
	}

	if err := normalizeCreate(&req); err != nil {
		t.Fatalf("normalizeCreate() error = %v", err)
	}
	if req.Region != "us-east-1" {
		t.Fatalf("region = %q, want us-east-1", req.Region)
	}
	if req.Name != "Recording archive" || req.Bucket != "recordings" {
		t.Fatalf("request was not normalized: %#v", req)
	}
}

func TestNormalizeCreateRejectsHTTPStorageEndpoint(t *testing.T) {
	req := CreateRequest{
		Name:            "Recording archive",
		EndpointURL:     "http://s3.example.com",
		Bucket:          "recordings",
		AccessKeyID:     "access",
		SecretAccessKey: "secret",
	}

	if err := normalizeCreate(&req); err == nil {
		t.Fatal("normalizeCreate() error = nil, want HTTPS endpoint error")
	}
}

func TestNormalizeCreateRejectsEndpointPath(t *testing.T) {
	req := CreateRequest{
		Name:            "Recording archive",
		EndpointURL:     "https://s3.example.com/path",
		Bucket:          "recordings",
		AccessKeyID:     "access",
		SecretAccessKey: "secret",
	}

	if err := normalizeCreate(&req); err == nil {
		t.Fatal("normalizeCreate() error = nil, want origin-only endpoint error")
	}
}

func TestNormalizeCreateRejectsPrivateEndpoint(t *testing.T) {
	for _, endpoint := range []string{
		"https://127.0.0.1",
		"https://10.0.0.1",
		"https://169.254.169.254",
		"https://localhost",
	} {
		req := CreateRequest{
			Name:            "Recording archive",
			EndpointURL:     endpoint,
			Bucket:          "recordings",
			AccessKeyID:     "access",
			SecretAccessKey: "secret",
		}
		if err := normalizeCreate(&req); err == nil {
			t.Fatalf("normalizeCreate(%q) error = nil, want public host error", endpoint)
		}
	}
}

func TestNormalizeCreateAllowsPrivateEndpointForSelfHosted(t *testing.T) {
	for _, endpoint := range []string{
		"http://10.0.0.25:9000",
		"https://minio.internal.example",
		"http://localhost:9000",
	} {
		req := CreateRequest{
			Name:            "Recording archive",
			EndpointURL:     endpoint,
			Bucket:          "recordings",
			AccessKeyID:     "access",
			SecretAccessKey: "secret",
		}
		if err := normalizeCreateWithPolicy(&req, true); err != nil {
			t.Fatalf("normalizeCreateWithPolicy(%q) error = %v", endpoint, err)
		}
	}
}

func TestNormalizeUpdateRequiresField(t *testing.T) {
	if err := normalizeUpdate(&UpdateRequest{}); err == nil {
		t.Fatal("normalizeUpdate() error = nil, want empty update error")
	}
}

func TestNormalizeUpdateRejectsInvalidStatus(t *testing.T) {
	status := "paused"
	if err := normalizeUpdate(&UpdateRequest{Status: &status}); err == nil {
		t.Fatal("normalizeUpdate() error = nil, want invalid status error")
	}
}
