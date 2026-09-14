package telecom

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/leamout/leamout/internal/telecom/calls"
	"github.com/leamout/leamout/internal/telecom/carriers"
	"github.com/leamout/leamout/internal/telecom/conferences"
	"github.com/leamout/leamout/internal/telecom/numbers"
	"github.com/leamout/leamout/internal/telecom/realtime"
	"github.com/leamout/leamout/internal/telecom/recordings"
	"github.com/leamout/leamout/internal/telecom/sip_domains"
	"github.com/leamout/leamout/internal/telecom/subscribers"
	"github.com/leamout/leamout/internal/telecom/trunks"
	"github.com/leamout/leamout/internal/telecom/voice"
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

	calls.RegisterRoutes(
		router,
		module.Calls.Handler,
		organizationAccess("calls"),
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

	numbers.RegisterRoutes(
		router,
		module.Numbers.Handler,
		organizationAccess("numbers"),
		idempotency,
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

	carriers.RegisterRoutes(
		router,
		module.Carriers.Handler,
		organizationAccess("carriers"),
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
