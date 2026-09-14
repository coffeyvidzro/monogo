package nats

const (
	JobsStreamName   = "LEAMOUT_JOBS"
	EventsStreamName = "LEAMOUT_EVENTS"
	DLQStreamName    = "LEAMOUT_DLQ"

	JobsSubjectPrefix   = "leamout.job."
	EventsSubjectPrefix = "leamout.event."
	DLQSubjectPrefix    = "leamout.dlq."

	JobsSubject   = JobsSubjectPrefix + ">"
	EventsSubject = EventsSubjectPrefix + ">"
	DLQSubject    = DLQSubjectPrefix + ">"
)
