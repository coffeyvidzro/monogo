package tools

// Executor dispatches tool calls. Built-in and webhook execution will be
// implemented when AI call orchestration is attached to live calls.
type Executor struct {
	service *Service
}

func NewExecutor(service *Service) *Executor {
	return &Executor{service: service}
}
