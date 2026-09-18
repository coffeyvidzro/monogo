package identity

import (
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/identity/auth"
	"github.com/coffeyvidzro/monogo/internal/identity/session"
	"github.com/coffeyvidzro/monogo/internal/identity/users"
)

type Module struct {
	Auth    AuthModule
	Session SessionModule
	Users   UsersModule
}

type AuthModule struct {
	Repository *auth.Repository
	Service    *auth.Service
	Handler    *auth.Handler
}

type SessionModule struct {
	Repository *session.Repository
	Service    *session.Service
	Handler    *session.Handler
}

type UsersModule struct {
	Repository *users.Repository
	Service    *users.Service
	Handler    *users.Handler
}

func New(
	queries *sqlc.Queries,
	development bool,
	cookieDomain string,
) *Module {
	sessionRepository := session.NewRepository(queries)
	sessionService := session.NewService(sessionRepository)

	authRepository := auth.NewRepository(queries)
	authService := auth.NewService(authRepository)

	usersRepository := users.NewRepository(queries)
	usersService := users.NewService(usersRepository)

	return &Module{
		Auth: AuthModule{
			Repository: authRepository,
			Service:    authService,
			Handler: auth.NewHandler(
				authService,
				sessionService,
				development,
				cookieDomain,
			),
		},
		Session: SessionModule{
			Repository: sessionRepository,
			Service:    sessionService,
			Handler: session.NewHandler(
				sessionService,
				development,
				cookieDomain,
			),
		},
		Users: UsersModule{
			Repository: usersRepository,
			Service:    usersService,
			Handler:    users.NewHandler(usersService),
		},
	}
}
