package smpp

import (
	"testing"

	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutlv"
)

func TestHandleDeliverSMInbound(t *testing.T) {
	client := &Client{
		inbound:    make(chan Inbound, 1),
		deliveries: make(chan Delivery, 1),
		errors:     make(chan error, 1),
	}
	body := pdu.NewDeliverSM()
	fields := body.Fields()
	if err := fields.Set(pdufield.SourceAddr, "12025550101"); err != nil {
		t.Fatalf("set source address: %v", err)
	}
	if err := fields.Set(pdufield.DestinationAddr, "12025550100"); err != nil {
		t.Fatalf("set destination address: %v", err)
	}
	if err := fields.Set(pdufield.ShortMessage, []byte("hello")); err != nil {
		t.Fatalf("set short message: %v", err)
	}
	if err := fields.Set(pdufield.ESMClass, byte(0)); err != nil {
		t.Fatalf("set ESM class: %v", err)
	}
	client.handlePDU(body)
	select {
	case got := <-client.Inbound():
		if got.From != "12025550101" || got.To != "12025550100" || got.Text != "hello" {
			t.Fatalf("inbound = %#v", got)
		}
	default:
		t.Fatal("inbound message not emitted")
	}
}

func TestHandleDeliverSMTLVReceipt(t *testing.T) {
	client := &Client{
		inbound:    make(chan Inbound, 1),
		deliveries: make(chan Delivery, 1),
		errors:     make(chan error, 1),
	}
	body := pdu.NewDeliverSM()
	fields := body.Fields()
	if err := fields.Set(pdufield.ESMClass, byte(0x04)); err != nil {
		t.Fatalf("set ESM class: %v", err)
	}
	_ = body.TLVFields().Set(pdutlv.TagReceiptedMessageID, pdutlv.CString("provider-42"))
	_ = body.TLVFields().Set(pdutlv.TagMessageStateOption, byte(2))
	_ = body.TLVFields().Set(pdutlv.TagNetworkErrorCode, []byte{0, 0, 0})
	client.handlePDU(body)
	select {
	case got := <-client.Deliveries():
		if got.MessageID != "provider-42" || got.State != DeliveryDelivered || got.ErrorCode != "000000" {
			t.Fatalf("delivery = %#v", got)
		}
	default:
		t.Fatal("TLV delivery receipt not emitted")
	}
}

func TestHandleDeliverSMReceipt(t *testing.T) {
	client := &Client{
		inbound:    make(chan Inbound, 1),
		deliveries: make(chan Delivery, 1),
		errors:     make(chan error, 1),
	}
	body := pdu.NewDeliverSM()
	fields := body.Fields()
	if err := fields.Set(pdufield.ShortMessage, []byte("id:abc stat:DELIVRD err:000")); err != nil {
		t.Fatalf("set short message: %v", err)
	}
	if err := fields.Set(pdufield.ESMClass, byte(0x04)); err != nil {
		t.Fatalf("set ESM class: %v", err)
	}
	client.handlePDU(body)
	select {
	case got := <-client.Deliveries():
		if got.MessageID != "abc" || got.State != DeliveryDelivered {
			t.Fatalf("delivery = %#v", got)
		}
	default:
		t.Fatal("delivery receipt not emitted")
	}
}
