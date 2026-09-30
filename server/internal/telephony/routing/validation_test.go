package routing

import "testing"

func TestManagedDestinationNormalizesSupportedURIs(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "+12125551234", want: "+12125551234"},
		{input: "tel:+442071838750;phone-context=example.com", want: "+442071838750"},
		{input: "SIP:+12125551234@voice.example.com", want: "+12125551234"},
	} {
		normalized, digits, err := managedDestination(test.input)
		if err != nil {
			t.Fatalf("managedDestination(%q): %v", test.input, err)
		}
		if normalized != test.want || digits != test.want[1:] {
			t.Fatalf("managedDestination(%q) = %q, %q", test.input, normalized, digits)
		}
	}
}

func TestManagedDestinationRejectsNonE164Routes(t *testing.T) {
	for _, input := range []string{"", "2125551234", "sip:alice@example.com", "+01234567", "+123"} {
		if _, _, err := managedDestination(input); err == nil {
			t.Fatalf("managedDestination(%q) unexpectedly succeeded", input)
		}
	}
}

func TestOutboundDecisionPrimaryUsesFirstRankedRoute(t *testing.T) {
	decision := OutboundDecision{Routes: []OutboundRoute{{Host: "primary"}, {Host: "secondary"}}}
	primary, ok := decision.Primary()
	if !ok || primary.Host != "primary" {
		t.Fatalf("Primary() = %+v, %v", primary, ok)
	}
	if _, ok := (OutboundDecision{}).Primary(); ok {
		t.Fatal("empty decision returned a primary route")
	}
}
