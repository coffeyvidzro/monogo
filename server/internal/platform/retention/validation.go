package retention

import "github.com/coffeyvidzro/monogo/pkg/apperror"

func validateResource(resource string) error {
	switch resource {
	case ResourceRecordings, ResourceConversations, ResourceAuditEvents:
		return nil
	default:
		return apperror.NewBadRequest("unsupported retention resource")
	}
}
func validateUpsert(req UpsertRequest) error {
	if req.RetentionDays < 1 || req.RetentionDays > 3650 {
		return apperror.NewBadRequest("retention_days must be between 1 and 3650")
	}
	return nil
}
