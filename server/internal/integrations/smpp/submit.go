package smpp

import (
	"context"
	"errors"
	"fmt"

	runtime "github.com/coffeyvidzro/monogo/internal/runtime/messaging"
	gosmpp "github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
)

func (c *Client) Send(ctx context.Context, request runtime.Request) (runtime.Result, error) {
	if err := ctx.Err(); err != nil {
		return runtime.Result{}, err
	}
	message, err := c.transceiver.Submit(&gosmpp.ShortMessage{Src: request.From, Dst: request.To, Text: pdutext.Raw(request.Text), Register: pdufield.FinalDeliveryReceipt})
	if errors.Is(err, gosmpp.ErrNotConnected) {
		return runtime.Result{}, ErrNotConnected
	}
	if err != nil {
		return runtime.Result{}, fmt.Errorf("submit SMPP message: %w", err)
	}
	id := message.RespID()
	if id == "" {
		return runtime.Result{}, fmt.Errorf("SMPP submit response did not contain a message id")
	}
	return runtime.Result{ExternalID: id}, nil
}
