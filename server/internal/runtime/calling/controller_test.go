package calling

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestFreeSWITCHEgress(t *testing.T) {
	endpoint, routeURI, err := freeSWITCHEgress(OriginateRequest{
		Destination: "+14155550100",
		CallerID:    "+14155550199",
		Host:        "carrier.example.com",
		Port:        5061,
		Transport:   "tls",
	})
	if err != nil {
		t.Fatal(err)
	}
	if endpoint != "sofia/internal/+14155550100@opensips:5060;transport=udp" {
		t.Fatalf("endpoint = %q", endpoint)
	}
	if routeURI != "sip:carrier.example.com:5061;transport=tls" {
		t.Fatalf("route URI = %q", routeURI)
	}
}

func TestFreeSWITCHEgressRejectsUnsafeHost(t *testing.T) {
	_, _, err := freeSWITCHEgress(OriginateRequest{
		Destination: "+14155550100",
		Host:        "carrier.example.com\ninvalid",
		Port:        5060,
		Transport:   "udp",
	})
	if err == nil {
		t.Fatal("expected invalid host error")
	}
}

func TestEgressVariablesCarryLeamoutIdentity(t *testing.T) {
	callID := uuid.New()
	trunkID := uuid.New()

	variables, err := egressVariables(OriginateRequest{
		CallID:              callID,
		TrunkID: trunkID,
		Privacy:             true,
		DTMFMode:            "rfc2833",
		MediaEncryption:     "sdes_srtp",
		MaxDurationSeconds:  60,
	}, "sip:carrier.example.com:5061;transport=tls")
	if err != nil {
		t.Fatal(err)
	}

	if variables[leamoutCallIDVar] != callID.String() {
		t.Fatalf("call id variable = %q", variables[leamoutCallIDVar])
	}
	if variables[trunkHeaderVar] != trunkID.String() {
		t.Fatalf("trunk id variable = %q", variables[trunkHeaderVar])
	}
	if !strings.Contains(variables[routeURIHeaderVar], "carrier.example.com") {
		t.Fatalf("route URI variable = %q", variables[routeURIHeaderVar])
	}
	if variables["execute_on_answer"] != "sched_hangup +60 ALLOTTED_TIMEOUT" {
		t.Fatalf(
			"execute_on_answer variable = %q",
			variables["execute_on_answer"],
		)
	}
}

func TestEgressVariablesRejectNegativeMaximumDuration(t *testing.T) {
	_, err := egressVariables(
		OriginateRequest{
			CallID:              uuid.New(),
			TrunkID: uuid.New(),
			MaxDurationSeconds:  -1,
		},
		"sip:carrier.example.com:5061;transport=tls",
	)
	if err == nil {
		t.Fatal("expected maximum duration validation error")
	}
}
