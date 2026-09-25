package commercial

import "github.com/go-chi/chi/v5"

// RegisterRoutes is the commercial module's routing entry point. No public
// checkout, payment, wallet or usage HTTP contract exists in this PR.
// Internal ingestion must never be exposed by registering an unguarded route.
func RegisterRoutes(_ chi.Router, _ *Module) {}
