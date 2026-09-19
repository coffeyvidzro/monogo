package didww

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type VoiceInTrunk struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name                string `json:"name"`
		ExternalReferenceID string `json:"external_reference_id"`
		Configuration       struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Host string `json:"host"`
				Port *int   `json:"port"`
			} `json:"attributes"`
		} `json:"configuration"`
	} `json:"attributes"`
}

type VoiceInTrunkList struct {
	Data  []VoiceInTrunk `json:"data"`
	Meta  Meta           `json:"meta"`
	Links Links          `json:"links"`
}

type SIPTrunkRequest struct {
	Name                string
	ExternalReferenceID string
	Host                string
	Port                int
	CodecIDs            []int
}

func (c *Client) ListVoiceInTrunks(ctx context.Context, page int) (VoiceInTrunkList, error) {
	if page < 1 {
		return VoiceInTrunkList{}, fmt.Errorf("didww page must be positive")
	}
	var result VoiceInTrunkList
	if err := c.get(ctx, "/voice_in_trunks", url.Values{"page[number]": {strconv.Itoa(page)}}, &result); err != nil {
		return VoiceInTrunkList{}, err
	}
	return result, nil
}

func (c *Client) GetVoiceInTrunk(ctx context.Context, id string) (VoiceInTrunk, error) {
	path, err := resourcePath("/voice_in_trunks", id)
	if err != nil {
		return VoiceInTrunk{}, err
	}
	var result single[VoiceInTrunk]
	if err := c.get(ctx, path, nil, &result); err != nil {
		return VoiceInTrunk{}, err
	}
	return result.Data, nil
}

// CreateSIPVoiceInTrunk provisions a provider destination. The caller must
// restrict Host to an approved Leamout SIP ingress, never an arbitrary URL.
func (c *Client) CreateSIPVoiceInTrunk(ctx context.Context, request SIPTrunkRequest) (VoiceInTrunk, error) {
	if err := validateSIPTrunk(request); err != nil {
		return VoiceInTrunk{}, err
	}
	payload := map[string]any{"data": map[string]any{
		"type": "voice_in_trunks",
		"attributes": map[string]any{
			"name": request.Name,
			"external_reference_id": request.ExternalReferenceID,
			"configuration": map[string]any{
				"type": "sip_configurations",
				"attributes": map[string]any{"host": request.Host, "port": request.Port, "codec_ids": request.CodecIDs},
			},
		},
	}}
	var result single[VoiceInTrunk]
	if err := c.do(ctx, http.MethodPost, "/voice_in_trunks", nil, payload, &result); err != nil {
		return VoiceInTrunk{}, err
	}
	return result.Data, nil
}

func validateSIPTrunk(request SIPTrunkRequest) error {
	if strings.TrimSpace(request.Name) == "" || strings.TrimSpace(request.Host) == "" ||
		strings.ContainsAny(request.Host, " /\\\r\n") || strings.Contains(request.Host, "@") ||
		request.Port < 1 || request.Port > 65535 || len(request.CodecIDs) == 0 {
		return fmt.Errorf("didww SIP trunk requires a name, valid host, port and codecs")
	}
	for _, codec := range request.CodecIDs {
		switch codec {
		case 6, 7, 8, 9, 10, 12, 13, 14, 15, 16, 17, 18, 19:
		default:
			return fmt.Errorf("didww unsupported SIP codec ID")
		}
	}
	if len(request.ExternalReferenceID) > 100 {
		return fmt.Errorf("didww trunk reference exceeds 100 characters")
	}
	return nil
}

// AssignDIDVoiceInTrunk attaches an owned DID to an existing DIDWW trunk.
// The resulting DID relationship must be verified before local activation.
func (c *Client) AssignDIDVoiceInTrunk(ctx context.Context, didID, trunkID string) (DID, error) {
	path, err := resourcePath("/dids", didID)
	if err != nil {
		return DID{}, err
	}
	if _, err := resourcePath("/voice_in_trunks", trunkID); err != nil {
		return DID{}, err
	}
	payload := map[string]any{"data": map[string]any{
		"id": didID, "type": "dids",
		"relationships": map[string]any{
			"voice_in_trunk": map[string]any{"data": ResourceIdentifier{Type: "voice_in_trunks", ID: trunkID}},
			"voice_in_trunk_group": map[string]any{"data": nil},
		},
	}}
	var result single[DID]
	if err := c.do(ctx, http.MethodPatch, path, url.Values{"include": {"voice_in_trunk"}}, payload, &result); err != nil {
		return DID{}, err
	}
	return result.Data, nil
}

func (c *Client) DeleteVoiceInTrunk(ctx context.Context, id string) error {
	path, err := resourcePath("/voice_in_trunks", id)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodDelete, path, nil, nil, nil)
}
