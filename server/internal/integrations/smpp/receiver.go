package smpp

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutlv"
)

func (c *Client) handlePDU(body pdu.Body) {
	if body == nil || body.Header().ID != pdu.DeliverSMID {
		return
	}
	fields := body.Fields()
	text := fieldString(fields, pdufield.ShortMessage)
	esm := fieldByte(fields, pdufield.ESMClass)
	if esm&0x04 != 0 {
		delivery, err := parseDeliveryPDU(body, text)
		if err != nil {
			select {
			case c.errors <- err:
			default:
			}
			return
		}
		select {
		case c.deliveries <- delivery:
		default:
		}
		return
	}
	message := Inbound{
		From:       fieldString(fields, pdufield.SourceAddr),
		To:         fieldString(fields, pdufield.DestinationAddr),
		Text:       text,
		ReceivedAt: time.Now().UTC(),
	}
	if message.From == "" || message.To == "" {
		return
	}
	select {
	case c.inbound <- message:
	default:
	}
}

func parseDeliveryPDU(body pdu.Body, text string) (Delivery, error) {
	delivery, textErr := ParseDeliveryReceipt(text)
	tlv := body.TLVFields()
	id := tlv[pdutlv.TagReceiptedMessageID]
	state := tlv[pdutlv.TagMessageStateOption]
	if id == nil || state == nil || len(state.Bytes()) == 0 {
		return delivery, textErr
	}
	delivery.MessageID = strings.TrimRight(string(id.Bytes()), "\x00")
	delivery.State = deliveryStateCode(state.Bytes()[0])
	if networkError := tlv[pdutlv.TagNetworkErrorCode]; networkError != nil {
		delivery.ErrorCode = hex.EncodeToString(networkError.Bytes())
	}
	delivery.Raw = text
	if delivery.MessageID == "" || delivery.State == "" {
		return Delivery{}, ErrMalformedReceipt
	}
	return delivery, nil
}

func deliveryStateCode(code byte) DeliveryState {
	switch code {
	case 1:
		return DeliveryEnroute
	case 2:
		return DeliveryDelivered
	case 3:
		return DeliveryExpired
	case 4:
		return DeliveryDeleted
	case 5:
		return DeliveryUndeliverable
	case 6:
		return DeliveryAccepted
	case 7:
		return DeliveryUnknown
	case 8:
		return DeliveryRejected
	default:
		return ""
	}
}

func fieldString(fields pdufield.Map, name pdufield.Name) string {
	if value := fields[name]; value != nil {
		return value.String()
	}
	return ""
}
func fieldByte(fields pdufield.Map, name pdufield.Name) byte {
	if value := fields[name]; value != nil && len(value.Bytes()) > 0 {
		return value.Bytes()[0]
	}
	return 0
}
