package telephony

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/coffeyvidzro/monogo/internal/telephony/calls"
	"github.com/coffeyvidzro/monogo/internal/telephony/carriers"
	"github.com/coffeyvidzro/monogo/internal/telephony/conferences"
	"github.com/coffeyvidzro/monogo/internal/telephony/numbers"
	"github.com/coffeyvidzro/monogo/internal/telephony/recordings"
	"github.com/coffeyvidzro/monogo/internal/telephony/sip_domains"
	"github.com/coffeyvidzro/monogo/internal/telephony/subscribers"
	"github.com/coffeyvidzro/monogo/internal/telephony/trunks"
	"github.com/coffeyvidzro/monogo/internal/telephony/voice"
	"github.com/coffeyvidzro/monogo/internal/telephony/webrtc"
)

func RegisterRoutes(
	router chi.Router,
	module *Module,
	organizationAccess func(string) func(http.Handler) http.Handler,
	idempotency func(http.Handler) http.Handler,
) {
	calls.RegisterRoutes(
		router,
		module.Calls.Handler,
		organizationAccess("calls"),
		idempotency,
	)

	numbers.RegisterRoutes(
		router,
		module.Numbers.Handler,
		organizationAccess("numbers"),
		idempotency,
	)

	voice.RegisterRoutes(
		router,
		module.Voice.Handler,
		organizationAccess("voice-applications"),
	)
	carriers.RegisterRoutes(router, module.Carriers.Handler, organizationAccess("carriers"))

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

	webrtc.RegisterRoutes(
		router,
		module.WebRTC.Handler,
		organizationAccess("webrtc"),
	)
}
