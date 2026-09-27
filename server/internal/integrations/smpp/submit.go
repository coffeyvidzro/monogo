package smpp

import (
	"context"
	"errors"
	"fmt"

	gosmpp "github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
)

// Submit exposes SMPP-owned types; canonical translation belongs to runtime/messaging.
func (c *Client) Submit(ctx context.Context, request SubmitRequest) (SubmitResult, error) {
	if err := ctx.Err(); err != nil {
		return SubmitResult{}, err
	}
	message, err := c.transceiver.Submit(&gosmpp.ShortMessage{Src: request.From, Dst: request.To, Text: pdutext.Raw(request.Text), Register: pdufield.FinalDeliveryReceipt})
	if errors.Is(err, gosmpp.ErrNotConnected) {
		return SubmitResult{}, ErrNotConnected
	}
	if err != nil {
		return SubmitResult{}, fmt.Errorf("submit SMPP message: %w", err)
	}
	id := message.RespID()
	if id == "" {
		return SubmitResult{}, fmt.Errorf("SMPP submit response did not contain a message id")
	}
	return SubmitResult{MessageID: id}, nil
}
