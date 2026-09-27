package smpp

// DeliveryHandler receives provider-normalized delivery receipts.
type DeliveryHandler func(Delivery) error
