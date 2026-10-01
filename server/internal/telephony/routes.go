package telephony

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/coffeyvidzro/monogo/internal/telephony/calls"
	"github.com/coffeyvidzro/monogo/internal/telephony/carriers"
	"github.com/coffeyvidzro/monogo/internal/telephony/numbers"
	"github.com/coffeyvidzro/monogo/internal/telephony/recordings"
	"github.com/coffeyvidzro/monogo/internal/telephony/trunks"
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

	carriers.RegisterRoutes(router, module.Carriers.Handler, organizationAccess("carriers"))

	recordings.RegisterRoutes(
		router,
		module.Recordings.Handler,
		organizationAccess("recordings"),
	)



	trunks.RegisterRoutes(
		router,
		module.Trunks.Handler,
		organizationAccess("trunks"),
	)

	webrtc.RegisterRoutes(
		router,
		module.WebRTC.Handler,
		organizationAccess("webrtc"),
	)
}
