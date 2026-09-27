package whatsapp

import "fmt"

type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("WhatsApp API returned %d: %s", e.Status, e.Message)
}
