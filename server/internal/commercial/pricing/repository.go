package pricing

import "github.com/coffeyvidzro/monogo/internal/database/sqlc"

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{
		queries: queries,
	}
}
