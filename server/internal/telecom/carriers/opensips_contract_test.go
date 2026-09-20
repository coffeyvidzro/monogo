package carriers

import (
	"os"
	"strings"
	"testing"
)

func TestOpenSIPSBYOCAuthenticationContracts(t *testing.T) {
	migration := readContractFile(t, "../../../migrations/026_create_carrier_digest_credentials.sql")
	config := readContractFile(t, "../../../../containers/opensips/opensips.cfg")

	assertContains(t, migration, "CREATE VIEW opensips_outbound_carrier_credentials")
	assertContains(t, migration, "'0x' || d.ha1_md5 AS password")
	assertContains(t, migration, "CREATE VIEW opensips_inbound_carrier_credentials")
	assertContains(t, migration, "d.ha1_md5")
	assertContains(t, config, `proxy_authorize("", "opensips_inbound_carrier_credentials")`)
	assertContains(t, config, "FROM opensips_outbound_carrier_credentials")
	assertContains(t, config, "uac_auth()")
}

func TestOpenSIPSBYOCPathsFailClosed(t *testing.T) {
	config := readContractFile(t, "../../../../containers/opensips/opensips.cfg")
	assertContains(t, config, "WHERE (SELECT count(*) FROM resolved) = 1")
	assertContains(t, config, "Carrier Route Not Authorized")
	assertContains(t, config, "pn.carrier_connection_id = '$avp(carrier_connection_id)'::UUID")
	assertContains(t, config, "cc.organization_id = pn.organization_id")
}

func readContractFile(t *testing.T, path string) string {
	t.Helper()
	value, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(value)
}

func assertContains(t *testing.T, value, expected string) {
	t.Helper()
	if !strings.Contains(value, expected) {
		t.Fatalf("contract does not contain %q", expected)
	}
}
