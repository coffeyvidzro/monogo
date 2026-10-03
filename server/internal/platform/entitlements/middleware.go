package entitlements

import (
	"net/http"

	platformmiddleware "github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/httputil"
)

type Middleware struct {
	service *Service
}

func NewMiddleware(service *Service) *Middleware {
	return &Middleware{
		service: service,
	}
}

func (m *Middleware) Require(capability Capability) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			organizationID, ok := platformmiddleware.OrganizationIDFromContext(r.Context())
			if !ok {
				httputil.Error(w, apperror.NewBadRequest("organization context required"))
				return
			}
			if err := m.service.Require(r.Context(), organizationID, capability); err != nil {
				httputil.Error(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
