package messaging

// SMS marks the direct-carrier SMPP runtime dependency.
type SMS struct{ Sender }

func NewSMS(sender Sender) *SMS { return &SMS{Sender: sender} }
