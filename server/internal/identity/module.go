package identity

import (
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/identity/auth"
	"github.com/leamout/leamout/internal/identity/session"
	"github.com/leamout/leamout/internal/identity/users"
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

func New(queries *sqlc.Queries) *Module {
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
			Handler:    auth.NewHandler(authService, sessionService),
		},
		Session: SessionModule{
			Repository: sessionRepository,
			Service:    sessionService,
			Handler:    session.NewHandler(sessionService),
		},
		Users: UsersModule{
			Repository: usersRepository,
			Service:    usersService,
			Handler:    users.NewHandler(usersService),
		},
	}
}
