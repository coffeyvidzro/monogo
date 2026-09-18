package telecom

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/coffeyvidzro/monogo/internal/security/encryption"
	"github.com/coffeyvidzro/monogo/internal/telecom/calls"
	"github.com/coffeyvidzro/monogo/internal/telecom/carriers"
	"github.com/coffeyvidzro/monogo/internal/telecom/conferences"
	"github.com/coffeyvidzro/monogo/internal/telecom/realtime"
	"github.com/coffeyvidzro/monogo/internal/telecom/recordings"
	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
	"github.com/coffeyvidzro/monogo/internal/telecom/sip_domains"
	"github.com/coffeyvidzro/monogo/internal/telecom/subscribers"
	"github.com/coffeyvidzro/monogo/internal/telecom/trunks"
	"github.com/coffeyvidzro/monogo/internal/telecom/voice"
)

type Dependencies struct {
	DB                   *pgxpool.Pool
	Queries              *sqlc.Queries
	Redis                *redis.Client
	CallsController      calls.Controller
	ConferenceController conferences.Controller
	CredentialCipher     *encryption.Cipher
	RealtimeService      *realtime.Service
}

type Module struct {
	Calls       CallsModule
	Carriers    CarriersModule
	Conferences ConferencesModule
	Realtime    RealtimeModule
	Recordings  RecordingsModule
	Routing     RoutingModule
	SIPDomains  SIPDomainsModule
	Subscribers SubscribersModule
	Trunks      TrunksModule
	Voice       VoiceModule
}

type CallsModule struct {
	Repository *calls.Repository
	Service    *calls.Service
	Handler    *calls.Handler
}

type CarriersModule struct {
	Repository *carriers.Repository
	Service    *carriers.Service
	Handler    *carriers.Handler
}

type ConferencesModule struct {
	Repository *conferences.Repository
	Service    *conferences.Service
	Handler    *conferences.Handler
}

type RealtimeModule struct {
	Service *realtime.Service
	Handler *realtime.Handler
}

type RecordingsModule struct {
	Repository *recordings.Repository
	Service    *recordings.Service
	Handler    *recordings.Handler
}

type RoutingModule struct {
	Repository *routing.Repository
	Service    *routing.Service
}

type SIPDomainsModule struct {
	Repository *sip_domains.Repository
	Service    *sip_domains.Service
	Handler    *sip_domains.Handler
}

type SubscribersModule struct {
	Repository *subscribers.Repository
	Service    *subscribers.Service
	Handler    *subscribers.Handler
}

type TrunksModule struct {
	Repository *trunks.Repository
	Service    *trunks.Service
	Handler    *trunks.Handler
}

type VoiceModule struct {
	Repository *voice.Repository
	Service    *voice.Service
	Handler    *voice.Handler
}

func New(deps Dependencies) (*Module, error) {
	routingRepository := routing.NewRepository(deps.Queries)
	routingService := routing.NewService(routingRepository, nil)

	callsRepository := calls.NewRepository(deps.Queries)
	callsService := calls.NewService(
		callsRepository,
		routingService,
		deps.CallsController,
		calls.NewRedisChannelStore(deps.Redis),
		calls.NewRedisAdmissionLimiter(deps.Redis, callsRepository),
	)

	voiceRepository := voice.NewRepository(deps.Queries)
	voiceService := voice.NewService(voiceRepository)

	recordingsRepository := recordings.NewRepository(deps.DB)
	recordingsService := recordings.NewService(recordingsRepository, nil)

	conferencesRepository := conferences.NewRepository(deps.DB)
	conferencesService := conferences.NewService(
		conferencesRepository,
		deps.ConferenceController,
	)

	subscribersRepository := subscribers.NewRepository(deps.Queries)
	subscribersService := subscribers.NewService(subscribersRepository)

	sipDomainsRepository := sip_domains.NewRepository(deps.Queries)
	sipDomainsService := sip_domains.NewService(sipDomainsRepository)

	trunksRepository := trunks.NewRepository(deps.Queries)
	trunksService := trunks.NewService(trunksRepository, deps.DB)
	carriersRepository := carriers.NewRepository(deps.Queries)
	carriersService := carriers.NewService(carriersRepository, deps.DB, deps.CredentialCipher)

	return &Module{
		Calls: CallsModule{
			Repository: callsRepository,
			Service:    callsService,
			Handler:    calls.NewHandler(callsService),
		},
		Routing: RoutingModule{
			Repository: routingRepository,
			Service:    routingService,
		},
		Voice: VoiceModule{
			Repository: voiceRepository,
			Service:    voiceService,
			Handler:    voice.NewHandler(voiceService),
		},
		Recordings: RecordingsModule{
			Repository: recordingsRepository,
			Service:    recordingsService,
			Handler:    recordings.NewHandler(recordingsService),
		},
		Conferences: ConferencesModule{
			Repository: conferencesRepository,
			Service:    conferencesService,
			Handler:    conferences.NewHandler(conferencesService),
		},
		Subscribers: SubscribersModule{
			Repository: subscribersRepository,
			Service:    subscribersService,
			Handler:    subscribers.NewHandler(subscribersService),
		},
		SIPDomains: SIPDomainsModule{
			Repository: sipDomainsRepository,
			Service:    sipDomainsService,
			Handler:    sip_domains.NewHandler(sipDomainsService),
		},
		Carriers: CarriersModule{
			Repository: carriersRepository,
			Service:    carriersService,
			Handler:    carriers.NewHandler(carriersService),
		},
		Trunks: TrunksModule{
			Repository: trunksRepository,
			Service:    trunksService,
			Handler:    trunks.NewHandler(trunksService),
		},
		Realtime: RealtimeModule{
			Service: deps.RealtimeService,
			Handler: realtime.NewHandler(deps.RealtimeService),
		},
	}, nil
}
