package smpp

type SubmitRequest struct{ From, To, Text string }
type SubmitResult struct{ MessageID string }
type Delivery struct{ ExternalID, State, ErrorCode string }
type Inbound struct{ ExternalID, From, To, Text string }
