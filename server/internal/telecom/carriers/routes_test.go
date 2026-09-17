package carriers

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutesExposesCarrierConnectionLifecycle(t *testing.T) {
	router := chi.NewRouter()
	passthrough := func(next http.Handler) http.Handler { return next }
	RegisterRoutes(router, NewHandler(nil), passthrough)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/carrier-connections/"},
		{http.MethodGet, "/carrier-connections/"},
		{http.MethodGet, "/carrier-connections/8d86799d-e068-47ec-97f5-76322a676dcf"},
		{http.MethodPost, "/carrier-connections/8d86799d-e068-47ec-97f5-76322a676dcf/validate"},
		{http.MethodPatch, "/carrier-connections/8d86799d-e068-47ec-97f5-76322a676dcf"},
		{http.MethodDelete, "/carrier-connections/8d86799d-e068-47ec-97f5-76322a676dcf"},
		{http.MethodPut, "/carrier-connections/8d86799d-e068-47ec-97f5-76322a676dcf/outbound-auth"},
		{http.MethodDelete, "/carrier-connections/8d86799d-e068-47ec-97f5-76322a676dcf/outbound-auth"},
		{http.MethodPut, "/carrier-connections/8d86799d-e068-47ec-97f5-76322a676dcf/inbound-auth"},
		{http.MethodDelete, "/carrier-connections/8d86799d-e068-47ec-97f5-76322a676dcf/inbound-auth"},
		{http.MethodPost, "/carrier-connections/8d86799d-e068-47ec-97f5-76322a676dcf/source-ips"},
		{http.MethodGet, "/carrier-connections/8d86799d-e068-47ec-97f5-76322a676dcf/source-ips"},
		{
			http.MethodDelete,
			"/carrier-connections/8d86799d-e068-47ec-97f5-76322a676dcf/source-ips/554cc6c9-9b57-42ed-bacf-66a515bb0805",
		},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			context := chi.NewRouteContext()
			if !router.Match(context, route.method, route.path) {
				t.Fatalf("route %s %s is not registered", route.method, route.path)
			}
		})
	}
}
