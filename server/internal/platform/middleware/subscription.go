package middleware

import (
	"context"
	"net/http"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/httputil"
	"github.com/google/uuid"
)

type subscriptionEntitlement interface {
	RequireActive(context.Context, uuid.UUID) error
}

func RequireSubscription(service subscriptionEntitlement) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			organizationID, ok := OrganizationIDFromContext(r.Context())
			if !ok {
				httputil.Error(w, apperror.NewBadRequest("organization context required"))
				return
			}
			if err := service.RequireActive(r.Context(), organizationID); err != nil {
				httputil.Error(w, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
