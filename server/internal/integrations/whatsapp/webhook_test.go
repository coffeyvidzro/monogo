package whatsapp

import "testing"

func TestParseWebhookMessagesAndStatuses(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account","entry":[{"changes":[{"field":"messages","value":{"metadata":{"phone_number_id":"phone-1","display_phone_number":"+15550000"},"messages":[{"from":"15550001","id":"wamid.in","type":"text","text":{"body":"hello"}}],"statuses":[{"id":"wamid.out","status":"delivered","recipient_id":"15550002"}]}}]}]}`)
	events, err := ParseWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Kind != EventMessage || events[0].Text != "hello" || events[0].To != "+15550000" || events[1].Kind != EventStatus || events[1].Status != "delivered" {
		t.Fatalf("events = %#v", events)
	}
}
