package messaging

// WhatsApp marks the official Business Platform runtime dependency.
type WhatsApp struct{ Sender }

func NewWhatsApp(sender Sender) *WhatsApp { return &WhatsApp{Sender: sender} }
