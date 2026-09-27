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
type WebhookEvent struct{ ExternalID, From, To, Text, Status, ErrorCode string }
