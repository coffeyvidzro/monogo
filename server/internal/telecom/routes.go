package telecom

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/coffeyvidzro/monogo/internal/telecom/conferences"
	"github.com/coffeyvidzro/monogo/internal/telecom/realtime"
	"github.com/coffeyvidzro/monogo/internal/telecom/recordings"
	"github.com/coffeyvidzro/monogo/internal/telecom/sip_domains"
	"github.com/coffeyvidzro/monogo/internal/telecom/subscribers"
	"github.com/coffeyvidzro/monogo/internal/telecom/trunks"
	"github.com/coffeyvidzro/monogo/internal/telecom/voice"
)

func RegisterRoutes(
	router chi.Router,
	module *Module,
	organizationAccess func(string) func(http.Handler) http.Handler,
	idempotency func(http.Handler) http.Handler,
) {
	voice.RegisterRoutes(
		router,
		module.Voice.Handler,
		organizationAccess("voice-applications"),
	)


	recordings.RegisterRoutes(
		router,
		module.Recordings.Handler,
		organizationAccess("recordings"),
	)

	subscribers.RegisterRoutes(
		router,
		module.Subscribers.Handler,
		organizationAccess("subscribers"),
	)

	sip_domains.RegisterRoutes(
		router,
		module.SIPDomains.Handler,
		organizationAccess("sip-domains"),
	)

	trunks.RegisterRoutes(
		router,
		module.Trunks.Handler,
		organizationAccess("trunks"),
	)

	conferences.RegisterRoutes(
		router,
		module.Conferences.Handler,
		organizationAccess("conferences"),
	)

	realtime.RegisterRoutes(
		router,
		module.Realtime.Handler,
		organizationAccess("realtime"),
	)
}
