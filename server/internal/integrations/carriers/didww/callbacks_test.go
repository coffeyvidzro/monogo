package didww

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"testing"
)

func TestOrderCallbackPOSTSignature(t *testing.T) {
	client, err := New(Config{APIKey: "callback-key"})
	if err != nil {
		t.Fatal(err)
	}
	callbackURL := "https://leamout.com/internal/didww/order"
	values := url.Values{"id": {"order-1"}, "type": {"orders"}, "status": {"completed"}}
	// Normalized URL includes the default HTTPS port. Fields are sorted by name.
	mac := hmac.New(sha1.New, []byte("callback-key"))
	_, _ = mac.Write([]byte("https://leamout.com:443/internal/didww/order" +
		"idorder-1statuscompletedtypeorders"))
	signature := hex.EncodeToString(mac.Sum(nil))
	event, err := client.ParseOrderCallbackPOST(callbackURL, values, signature)
	if err != nil || event.ID != "order-1" || event.Status != "completed" {
		t.Fatalf("callback = %+v, error = %v", event, err)
	}
	if _, err := client.ParseOrderCallbackPOST(callbackURL, values, "0000000000000000000000000000000000000000"); err == nil {
		t.Fatal("expected invalid signature error")
	}
	values.Set("status", "canceled")
	if _, err := client.ParseOrderCallbackPOST(callbackURL, values, signature); err == nil {
		t.Fatal("expected tampered callback rejection")
	}
}
