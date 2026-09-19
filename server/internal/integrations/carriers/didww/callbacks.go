package didww

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// OrderCallback is a verified DIDWW order-state notification, not an instruction
// to activate a local DID. Retrieve the order and DID independently.
type OrderCallback struct {
	ID     string
	Type   string
	Status string
}

// ParseOrderCallbackPOST verifies the documented DIDWW POST signature:
// HMAC-SHA1 over the normalized callback URL and sorted form field names/values.
// callbackURL must be the externally configured URL, not a forwarded internal URL.
func (c *Client) ParseOrderCallbackPOST(callbackURL string, form url.Values, signature string) (OrderCallback, error) {
	if len(signature) != 40 {
		return OrderCallback{}, fmt.Errorf("didww callback signature is invalid")
	}
	got, err := hex.DecodeString(signature)
	if err != nil || len(got) != sha1.Size || signature != strings.ToLower(signature) {
		return OrderCallback{}, fmt.Errorf("didww callback signature is invalid")
	}
	u, err := url.Parse(callbackURL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return OrderCallback{}, fmt.Errorf("didww callback URL is invalid")
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	userinfo := ""
	if u.User != nil {
		userinfo = u.User.String() + "@"
	}
	normalized := u.Scheme + "://" + userinfo + u.Hostname() + ":" + port + u.EscapedPath()
	if u.RawQuery != "" {
		normalized += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		normalized += "#" + u.Fragment
	}
	keys := make([]string, 0, len(form))
	for key, values := range form {
		if len(values) != 1 {
			return OrderCallback{}, fmt.Errorf("didww callback field is invalid")
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var signed strings.Builder
	signed.WriteString(normalized)
	for _, key := range keys {
		signed.WriteString(key)
		signed.WriteString(form.Get(key))
	}
	mac := hmac.New(sha1.New, []byte(c.apiKey))
	_, _ = mac.Write([]byte(signed.String()))
	if !hmac.Equal(got, mac.Sum(nil)) {
		return OrderCallback{}, fmt.Errorf("didww callback signature is invalid")
	}
	event := OrderCallback{ID: form.Get("id"), Type: form.Get("type"), Status: form.Get("status")}
	if event.ID == "" || event.Type != "orders" || (event.Status != "completed" && event.Status != "canceled") || len(form) != 3 {
		return OrderCallback{}, fmt.Errorf("didww callback payload is invalid")
	}
	return event, nil
}
