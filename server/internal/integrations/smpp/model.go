package smpp

type Delivery struct{ ExternalID, State, ErrorCode string }
type Inbound struct{ ExternalID, From, To, Text string }
