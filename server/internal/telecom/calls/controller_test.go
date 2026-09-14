package calls

import (
	"testing"

	"github.com/google/uuid"
)

func TestFreeSWITCHEgress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		req          OriginateRequest
		wantEndpoint string
		wantRouteURI string
		wantErr      bool
	}{
		{
			name: "udp route",
			req: OriginateRequest{
				Host:        "sip.carrier.example",
				Port:        5060,
				Transport:   "udp",
				Destination: "+14155550100",
			},
			wantEndpoint: "sofia/internal/+14155550100@opensips:5060;transport=udp",
			wantRouteURI: "sip:sip.carrier.example:5060;transport=udp",
		},
		{
			name: "ipv6 route",
			req: OriginateRequest{
				Host:        "2001:db8::10",
				Port:        5061,
				Transport:   "TLS",
				Destination: "+233200000000",
			},
			wantEndpoint: "sofia/internal/+233200000000@opensips:5060;transport=udp",
			wantRouteURI: "sip:[2001:db8::10]:5061;transport=tls",
		},
		{
			name: "invalid transport",
			req: OriginateRequest{
				Host:        "sip.carrier.example",
				Port:        5060,
				Transport:   "ws",
				Destination: "+14155550100",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			endpoint, routeURI, err := freeSWITCHEgress(tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("freeSWITCHEgress() error = %v, wantErr %t", err, tt.wantErr)
			}
			if endpoint != tt.wantEndpoint {
				t.Fatalf("freeSWITCHEgress() endpoint = %q, want %q", endpoint, tt.wantEndpoint)
			}
			if routeURI != tt.wantRouteURI {
				t.Fatalf("freeSWITCHEgress() route URI = %q, want %q", routeURI, tt.wantRouteURI)
			}
		})
	}
}

func TestEgressVariablesBindResolvedCarrierConnection(t *testing.T) {
	t.Parallel()

	connectionID := uuid.New()
	variables, err := egressVariables(OriginateRequest{
		CarrierConnectionID: connectionID,
		Privacy:             true,
		DTMFMode:            "info",
		Codecs:              []string{"PCMU", "OPUS"},
		MediaEncryption:     "sdes_srtp",
		Variables: map[string]string{
			routeURIHeaderVar:          "sip:attacker.invalid",
			carrierConnectionHeaderVar: uuid.NewString(),
		},
	}, "sip:carrier.example:5060;transport=udp")
	if err != nil {
		t.Fatalf("egressVariables() error = %v", err)
	}
	if got := variables[routeURIHeaderVar]; got != "sip:carrier.example:5060;transport=udp" {
		t.Fatalf("route metadata = %q", got)
	}
	if got := variables[carrierConnectionHeaderVar]; got != connectionID.String() {
		t.Fatalf("carrier connection metadata = %q, want %q", got, connectionID)
	}
	if variables[privacyHeaderVar] != "id" || variables[dtmfTypeVar] != "info" || variables[codecStringVar] != "PCMU,OPUS" || variables[mediaEncryptionHeaderVar] != "sdes_srtp" {
		t.Fatalf("unexpected protected media variables: %+v", variables)
	}

	if _, err := egressVariables(OriginateRequest{}, "sip:carrier.example"); err == nil {
		t.Fatal("egressVariables() accepted an empty carrier connection id")
	}
}
