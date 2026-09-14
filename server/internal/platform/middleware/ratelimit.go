package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/ulule/limiter/v3"

	"github.com/leamout/leamout/internal/security/authn"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/httputil"
)

// RateLimitMiddleware enforces shared organization and credential request
// budgets. It is applied after authentication and organization resolution so
// callers cannot choose or spoof either rate-limit identity.
type RateLimitMiddleware struct {
	store limiter.Store
}

func NewRateLimitMiddleware(store limiter.Store) (*RateLimitMiddleware, error) {
	if store == nil {
		return nil, fmt.Errorf("rate limit store is required")
	}

	return &RateLimitMiddleware{store: store}, nil
}

// Handle separates ordinary reads and writes into independent budgets.
func (m *RateLimitMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			m.handle(
				next,
				"read",
				limiter.Rate{Period: time.Minute, Limit: 1200},
				limiter.Rate{Period: time.Minute, Limit: 600},
			).ServeHTTP(w, r)
			return
		}

		m.handle(
			next,
			"write",
			limiter.Rate{Period: time.Minute, Limit: 600},
			limiter.Rate{Period: time.Minute, Limit: 300},
		).ServeHTTP(w, r)
	})
}

func (m *RateLimitMiddleware) handle(
	next http.Handler,
	class string,
	organizationRate limiter.Rate,
	credentialRate limiter.Rate,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		organizationID, ok := OrganizationIDFromContext(r.Context())
		if !ok {
			httputil.Error(w, apperror.NewUnauthorized("organization context required"))
			return
		}
		principal, ok := authn.PrincipalFromContext(r.Context())
		if !ok || principal.Credential.ID == uuid.Nil {
			httputil.Error(w, apperror.NewUnauthorized("authentication required"))
			return
		}

		organizationLimit, err := limiter.New(m.store, organizationRate).Get(
			r.Context(),
			"http:organization:"+organizationID.String()+":"+class,
		)
		if err != nil {
			httputil.Error(w, apperror.NewServiceUnavailable("rate limit service unavailable", err))
			return
		}
		if organizationLimit.Reached {
			writeRateLimitResponse(w, organizationLimit, "organization rate limit exceeded")
			return
		}

		credentialLimit, err := limiter.New(m.store, credentialRate).Get(
			r.Context(),
			"http:organization:"+organizationID.String()+":credential:"+principal.Credential.ID.String()+":"+class,
		)
		if err != nil {
			httputil.Error(w, apperror.NewServiceUnavailable("rate limit service unavailable", err))
			return
		}
		if credentialLimit.Reached {
			writeRateLimitResponse(w, credentialLimit, "credential rate limit exceeded")
			return
		}

		if credentialLimit.Remaining*organizationLimit.Limit < organizationLimit.Remaining*credentialLimit.Limit {
			writeRateLimitHeaders(w, credentialLimit)
		} else {
			writeRateLimitHeaders(w, organizationLimit)
		}

		next.ServeHTTP(w, r)
	})
}

func writeRateLimitResponse(w http.ResponseWriter, limit limiter.Context, message string) {
	writeRateLimitHeaders(w, limit)
	retryAfter := limit.Reset - time.Now().Unix()
	if retryAfter < 1 {
		retryAfter = 1
	}
	w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
	httputil.Error(w, apperror.NewTooManyRequests(message))
}

func writeRateLimitHeaders(w http.ResponseWriter, limit limiter.Context) {
	w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(limit.Limit, 10))
	w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(limit.Remaining, 10))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(limit.Reset, 10))
}
