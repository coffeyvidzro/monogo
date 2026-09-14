package hasher

import "testing"

func TestComputeHA1(t *testing.T) {
	hashes := ComputeHA1("alice", "sip.example.com", "secret")
	if hashes.MD5 != "a0bbf6034b8565747c15ee9850d9215a" {
		t.Fatalf("unexpected MD5 HA1: %s", hashes.MD5)
	}
	if !VerifyHA1(hashes.SHA256, hashes.SHA256) || VerifyHA1(hashes.SHA256, hashes.MD5) {
		t.Fatal("HA1 comparison failed")
	}
}
