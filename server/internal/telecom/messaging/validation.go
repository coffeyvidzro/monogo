package messaging

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

const (
	maxAddressLength = 255
	maxBodyLength    = 10000
	maxMediaItems    = 10
)

func normalizeCreateRequest(req CreateRequest) (CreateRequest, error) {
	req.Channel = Channel(strings.ToLower(strings.TrimSpace(string(req.Channel))))
	req.From = strings.TrimSpace(req.From)
	req.To = strings.TrimSpace(req.To)
	if req.Body != nil {
		body := strings.TrimSpace(*req.Body)
		req.Body = &body
	}
	for i := range req.Media {
		req.Media[i].URL = strings.TrimSpace(req.Media[i].URL)
		req.Media[i].ContentType = strings.TrimSpace(req.Media[i].ContentType)
	}
	if err := validateMessagePayload(req.Channel, req.From, req.To, req.Body, req.Media); err != nil {
		return CreateRequest{}, err
	}
	return req, nil
}

func normalizeInboundRequest(req InboundRequest) (InboundRequest, error) {
	if req.OrganizationID == uuid.Nil {
		return InboundRequest{}, apperror.NewBadRequest("organization_id is required")
	}
	if req.CarrierConnectionID == uuid.Nil {
		return InboundRequest{}, apperror.NewBadRequest("carrier_connection_id is required")
	}
	req.ProviderMessageID = strings.TrimSpace(req.ProviderMessageID)
	if req.ProviderMessageID == "" || len(req.ProviderMessageID) > 255 {
		return InboundRequest{}, apperror.NewBadRequest("provider_message_id must be between 1 and 255 characters")
	}
	req.Channel = Channel(strings.ToLower(strings.TrimSpace(string(req.Channel))))
	req.From = strings.TrimSpace(req.From)
	req.To = strings.TrimSpace(req.To)
	if req.Body != nil {
		body := strings.TrimSpace(*req.Body)
		req.Body = &body
	}
	for i := range req.Media {
		req.Media[i].URL = strings.TrimSpace(req.Media[i].URL)
		req.Media[i].ContentType = strings.TrimSpace(req.Media[i].ContentType)
	}
	if err := validateMessagePayload(req.Channel, req.From, req.To, req.Body, req.Media); err != nil {
		return InboundRequest{}, err
	}
	return req, nil
}

func validateMessagePayload(channel Channel, from, to string, body *string, media []Media) error {
	if !channel.IsValid() {
		return apperror.NewBadRequest("unsupported message channel")
	}
	if from == "" || len(from) > maxAddressLength {
		return apperror.NewBadRequest("from must be between 1 and 255 characters")
	}
	if to == "" || len(to) > maxAddressLength {
		return apperror.NewBadRequest("to must be between 1 and 255 characters")
	}
	if body != nil && len(*body) > maxBodyLength {
		return apperror.NewBadRequest("body must not exceed 10000 characters")
	}
	if len(media) > maxMediaItems {
		return apperror.NewBadRequest("media must not contain more than 10 items")
	}
	if (body == nil || strings.TrimSpace(*body) == "") && len(media) == 0 {
		return apperror.NewBadRequest("body or media is required")
	}
	for _, item := range media {
		if err := validateMedia(item); err != nil {
			return err
		}
	}
	if channel == ChannelSMS && len(media) > 0 {
		return apperror.NewBadRequest("sms does not support media")
	}
	return nil
}

func validateMedia(item Media) error {
	parsed, err := url.Parse(item.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return apperror.NewBadRequest("media url must be an absolute HTTPS URL")
	}
	if len(item.ContentType) > 255 {
		return apperror.NewBadRequest("media content_type must not exceed 255 characters")
	}
	return nil
}

func (c Channel) IsValid() bool {
	switch c {
	case ChannelSMS, ChannelMMS, ChannelWhatsApp:
		return true
	default:
		return false
	}
}

func normalizeListRequest(req ListRequest) ListRequest {
	if req.Status != nil {
		value := strings.ToLower(strings.TrimSpace(*req.Status))
		req.Status = &value
	}
	if req.Direction != nil {
		value := strings.ToLower(strings.TrimSpace(*req.Direction))
		req.Direction = &value
	}
	if req.Channel != nil {
		value := strings.ToLower(strings.TrimSpace(*req.Channel))
		req.Channel = &value
	}
	return req
}

func validateListRequest(req ListRequest) error {
	if req.Offset < 0 {
		return apperror.NewBadRequest("offset cannot be negative")
	}
	if req.Limit < 1 || req.Limit > 100 {
		return apperror.NewBadRequest("limit must be between 1 and 100")
	}
	if req.Status != nil {
		status := Status(strings.ToLower(strings.TrimSpace(*req.Status)))
		if !status.IsValid() {
			return apperror.NewBadRequest("invalid message status")
		}
	}
	if req.Direction != nil {
		direction := Direction(strings.ToLower(strings.TrimSpace(*req.Direction)))
		if direction != DirectionInbound && direction != DirectionOutbound {
			return apperror.NewBadRequest("invalid message direction")
		}
	}
	if req.Channel != nil {
		channel := Channel(strings.ToLower(strings.TrimSpace(*req.Channel)))
		if !channel.IsValid() {
			return apperror.NewBadRequest("invalid message channel")
		}
	}
	return nil
}

func (s Status) IsValid() bool {
	switch s {
	case StatusQueued, StatusSubmitted, StatusSent, StatusDelivered, StatusUndelivered, StatusReceived, StatusFailed:
		return true
	default:
		return false
	}
}

func marshalMedia(media []Media) ([]byte, error) {
	if media == nil {
		media = []Media{}
	}
	value, err := json.Marshal(media)
	if err != nil {
		return nil, apperror.NewBadRequest("invalid media")
	}
	return value, nil
}
