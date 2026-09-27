package whatsapp

type sendRequest struct {
	MessagingProduct string      `json:"messaging_product"`
	To               string      `json:"to"`
	Type             string      `json:"type"`
	Text             textPayload `json:"text"`
}
type textPayload struct {
	Body string `json:"body"`
}
type sendResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}
type MessageRequest struct{ To, Text string }
type MessageResult struct{ MessageID string }
type EventKind string

const (
	EventMessage EventKind = "message"
	EventStatus  EventKind = "status"
)

type WebhookEvent struct {
	Kind                                                                      EventKind
	PhoneNumberID, MessageID, From, To, Text, Status, ErrorCode, ErrorMessage string
}
type webhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				Metadata struct {
					PhoneNumberID string `json:"phone_number_id"`
				} `json:"metadata"`
				Messages []struct {
					From string `json:"from"`
					ID   string `json:"id"`
					Type string `json:"type"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
				Statuses []struct {
					ID          string `json:"id"`
					Status      string `json:"status"`
					RecipientID string `json:"recipient_id"`
					Errors      []struct {
						Code    int    `json:"code"`
						Message string `json:"message"`
					} `json:"errors"`
				} `json:"statuses"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}
