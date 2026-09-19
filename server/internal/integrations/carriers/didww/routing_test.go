package didww

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTrunkAndAssignDID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", jsonAPIMediaType)
		switch r.URL.Path {
		case "/v3/voice_in_trunks":
			if r.Method != http.MethodPost {
				t.Errorf("trunk creation method = %s", r.Method)
			}
			var doc struct {
				Data struct {
					Type string `json:"type"`
					Attributes struct {
						Name string `json:"name"`
						Configuration struct {
							Type string `json:"type"`
							Attributes struct {
								Host string `json:"host"`
								Port int `json:"port"`
								CodecIDs []int `json:"codec_ids"`
							} `json:"attributes"`
						} `json:"configuration"`
					} `json:"attributes"`
				} `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
				t.Error(err)
			}
			cfg := doc.Data.Attributes.Configuration
			if doc.Data.Type != "voice_in_trunks" || doc.Data.Attributes.Name != "Leamout SIP" ||
				cfg.Type != "sip_configurations" || cfg.Attributes.Host != "sip.leamout.com" ||
				cfg.Attributes.Port != 5060 || len(cfg.Attributes.CodecIDs) != 2 {
				t.Errorf("unexpected trunk request: %+v", doc.Data)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":"trunk-1","type":"voice_in_trunks","attributes":{"name":"Leamout SIP"}}}`))
		case "/v3/dids/did-1":
			if r.Method != http.MethodPatch {
				t.Errorf("DID routing method = %s", r.Method)
			}
			var doc struct {
				Data struct {
					Relationships map[string]struct {
						Data *ResourceIdentifier `json:"data"`
					} `json:"relationships"`
				} `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
				t.Error(err)
			}
			rel := doc.Data.Relationships["voice_in_trunk"].Data
			if rel == nil || rel.Type != "voice_in_trunks" || rel.ID != "trunk-1" {
				t.Errorf("unexpected routing relationship: %+v", rel)
			}
			_, _ = w.Write([]byte(`{"data":{"id":"did-1","type":"dids","relationships":{"voice_in_trunk":{"data":{"type":"voice_in_trunks","id":"trunk-1"}}}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "key", BaseURL: server.URL + "/v3"})
	if err != nil {
		t.Fatal(err)
	}
	trunk, err := client.CreateSIPVoiceInTrunk(context.Background(), SIPTrunkRequest{
		Name: "Leamout SIP", Host: "sip.leamout.com", Port: 5060, CodecIDs: []int{9, 10},
	})
	if err != nil || trunk.ID != "trunk-1" {
		t.Fatalf("trunk = %+v, error = %v", trunk, err)
	}
	did, err := client.AssignDIDVoiceInTrunk(context.Background(), "did-1", trunk.ID)
	if err != nil || did.Relationships["voice_in_trunk"].Data == nil ||
		did.Relationships["voice_in_trunk"].Data.ID != trunk.ID {
		t.Fatalf("routed DID = %+v, error = %v", did, err)
	}
}
