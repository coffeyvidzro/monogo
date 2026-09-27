package smpp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	gosmpp "github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
)

func (c *Client) Submit(ctx context.Context, request SubmitRequest) (SubmitResult, error) {
	if err := ctx.Err(); err != nil {
		return SubmitResult{}, err
	}
	if strings.TrimSpace(request.From) == "" || strings.TrimSpace(request.To) == "" {
		return SubmitResult{}, fmt.Errorf("SMPP source and destination are required")
	}
	if request.Text == "" {
		return SubmitResult{}, fmt.Errorf("SMPP message text is required")
	}
	if c.submitter == nil {
		return SubmitResult{}, ErrSubmitUnsupported
	}
	state, _ := c.State()
	if state != StateConnected {
		return SubmitResult{}, ErrNotConnected
	}
	register := pdufield.NoDeliveryReceipt
	if request.RequestDeliveryReceipt {
		register = pdufield.FinalDeliveryReceipt
	}
	message, err := c.submitter.Submit(&gosmpp.ShortMessage{
		Src:           request.From,
		Dst:           request.To,
		Text:          pdutext.Raw(request.Text),
		Register:      register,
		ServiceType:   request.ServiceType,
		SourceAddrTON: request.SourceTON,
		SourceAddrNPI: request.SourceNPI,
		DestAddrTON:   request.DestinationTON,
		DestAddrNPI:   request.DestinationNPI,
	})
	if errors.Is(err, gosmpp.ErrNotConnected) {
		return SubmitResult{}, ErrNotConnected
	}
	if err != nil {
		return SubmitResult{}, fmt.Errorf("submit SMPP message: %w", err)
	}
	id := strings.TrimSpace(message.RespID())
	if id == "" {
		return SubmitResult{}, fmt.Errorf("SMPP submit_sm_resp did not contain a message id")
	}
	return SubmitResult{
		MessageID: id,
	}, nil
}
