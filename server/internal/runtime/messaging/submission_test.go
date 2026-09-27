package messaging

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/smpp"
	"github.com/coffeyvidzro/monogo/internal/integrations/whatsapp"
	"github.com/google/uuid"
)

func TestSMSSendClassifiesDisconnectedAsNotSubmitted(t *testing.T) {
	client, err := smpp.New(smpp.DefaultConfig("127.0.0.1", 2775, "system", "secret"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewSMS(client).Send(t.Context(), Request{
		MessageID:    uuid.New(),
		ConnectionID: uuid.New(),
		Channel:      ChannelSMS,
		From:         "233200000001",
		To:           "233200000002",
		Text:         "hello",
	})
	var submissionError *SubmissionError
	if !errors.As(err, &submissionError) {
		t.Fatalf("Send() error = %v", err)
	}
	if submissionError.Outcome != SubmissionNotSubmitted {
		t.Fatalf("outcome = %q", submissionError.Outcome)
	}
}

func TestWhatsAppSendClassifiesAPIRejection(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error":{"message":"rejected"}}`)
	}))
	defer server.Close()

	client, err := whatsapp.New(whatsapp.Config{
		BaseURL:       server.URL,
		AccessToken:   "token",
		PhoneNumberID: "phone",
		AppSecret:     "secret",
		Timeout:       time.Second,
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewWhatsApp(client).Send(t.Context(), Request{
		MessageID:    uuid.New(),
		ConnectionID: uuid.New(),
		Channel:      ChannelWhatsApp,
		To:           "233200000002",
		Text:         "hello",
	})
	var submissionError *SubmissionError
	if !errors.As(err, &submissionError) {
		t.Fatalf("Send() error = %v", err)
	}
	if submissionError.Outcome != SubmissionRejected {
		t.Fatalf("outcome = %q", submissionError.Outcome)
	}
}

func TestWhatsAppSendClassifiesTransportFailureAsUnknown(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	clientHTTP := server.Client()
	baseURL := server.URL
	server.Close()

	client, err := whatsapp.New(whatsapp.Config{
		BaseURL:       baseURL,
		AccessToken:   "token",
		PhoneNumberID: "phone",
		AppSecret:     "secret",
		Timeout:       time.Second,
	}, clientHTTP)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewWhatsApp(client).Send(t.Context(), Request{
		MessageID:    uuid.New(),
		ConnectionID: uuid.New(),
		Channel:      ChannelWhatsApp,
		To:           "233200000002",
		Text:         "hello",
	})
	var submissionError *SubmissionError
	if !errors.As(err, &submissionError) {
		t.Fatalf("Send() error = %v", err)
	}
	if submissionError.Outcome != SubmissionUnknown {
		t.Fatalf("outcome = %q", submissionError.Outcome)
	}
}
