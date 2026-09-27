package smpp

// InboundHandler receives provider-normalized mobile-originated messages.
type InboundHandler func(Inbound) error
