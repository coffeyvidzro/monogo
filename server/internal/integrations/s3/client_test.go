package s3

import "testing"

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		host   string
		secure bool
		ok     bool
	}{
		{name: "https", value: "https://s3.example.com", host: "s3.example.com", secure: true, ok: true},
		{name: "http", value: "http://minio:9000", host: "minio:9000", secure: false, ok: true},
		{name: "path", value: "https://s3.example.com/path", ok: false},
		{name: "credentials", value: "https://user:secret@s3.example.com", ok: false},
		{name: "query", value: "https://s3.example.com?x=1", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, secure, err := parseEndpoint(tt.value)
			if !tt.ok {
				if err == nil {
					t.Fatal("parseEndpoint() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseEndpoint() error = %v", err)
			}
			if host != tt.host || secure != tt.secure {
				t.Fatalf("parseEndpoint() = (%q, %v), want (%q, %v)", host, secure, tt.host, tt.secure)
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	for _, key := range []string{"", "/recording.wav", "../recording.wav", "recordings/../recording.wav"} {
		if err := validateKey(key); err == nil {
			t.Fatalf("validateKey(%q) error = nil, want error", key)
		}
	}
	if err := validateKey("recordings/org/recording.wav"); err != nil {
		t.Fatalf("validateKey() error = %v", err)
	}
}
