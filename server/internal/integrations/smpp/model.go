package smpp

import "time"

type SubmitRequest struct {
	From, To, Text                                       string
	ServiceType                                          string
	SourceTON, SourceNPI, DestinationTON, DestinationNPI uint8
	RequestDeliveryReceipt                               bool
}
type SubmitResult struct{ MessageID string }
type ConnectionState string

const (
	StateDisconnected ConnectionState = "disconnected"
	StateConnecting   ConnectionState = "connecting"
	StateConnected    ConnectionState = "connected"
	StateFailed       ConnectionState = "failed"
	StateClosed       ConnectionState = "closed"
)

type StateChange struct {
	State ConnectionState
	Err   error
	At    time.Time
}
type Inbound struct {
	From, To, Text string
	Reference      string
	ReceivedAt     time.Time
}
type DeliveryState string

const (
	DeliveryEnroute       DeliveryState = "enroute"
	DeliveryDelivered     DeliveryState = "delivered"
	DeliveryExpired       DeliveryState = "expired"
	DeliveryDeleted       DeliveryState = "deleted"
	DeliveryUndeliverable DeliveryState = "undeliverable"
	DeliveryAccepted      DeliveryState = "accepted"
	DeliveryUnknown       DeliveryState = "unknown"
	DeliveryRejected      DeliveryState = "rejected"
)

type Delivery struct {
	MessageID           string
	State               DeliveryState
	ErrorCode           string
	SubmittedAt, DoneAt *time.Time
	Raw                 string
}
