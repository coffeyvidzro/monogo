package commpeak

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ListTerminationCDRs returns the upstream JSON unchanged until CommPeak's
// account-specific CDR response schema has been verified against real fixtures.
// Never rate or charge customers directly from this integration response.
func (c *Client) ListTerminationCDRs(
	ctx context.Context,
	query TerminationCDRQuery,
) (json.RawMessage, error) {
	if err := query.validate(); err != nil {
		return nil, err
	}
	page, perPage := query.Page, query.PerPage
	if page == 0 {
		page = 1
	}
	if perPage == 0 {
		perPage = 100
	}
	values := url.Values{
		"time_range": {fmt.Sprintf("%s - %s", query.From.UTC().Format("2006-01-02"), query.To.UTC().Format("2006-01-02"))},
		"page":       {strconv.Itoa(page)},
		"per_page":   {strconv.Itoa(perPage)},
	}
	if id := strings.TrimSpace(query.SIPAccountID); id != "" {
		values.Set("sip_account_id", id)
	}
	return c.get(ctx, "/call_records/termination", values)
}
