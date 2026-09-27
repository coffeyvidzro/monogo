package settlement

type Outcome string

const (
	OutcomeApplied        Outcome = "applied"
	OutcomeAlreadyApplied Outcome = "already_applied"
	OutcomeRetry          Outcome = "retry"
	OutcomeVersionGap     Outcome = "version_gap"
	OutcomeIntegrity      Outcome = "integrity_conflict"
)

type Result struct {
	Outcome Outcome
}

type PersistenceError struct {
	Outcome Outcome
	Cause   error
}

func (e *PersistenceError) Error() string {
	return e.Cause.Error()
}

func (e *PersistenceError) Unwrap() error {
	return e.Cause
}
