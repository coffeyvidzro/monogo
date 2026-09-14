package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/leamout/leamout/internal/modules/idempotency"
	"github.com/leamout/leamout/internal/security/authn"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/httputil"
)

const maxIdempotentRequestBody = 1 << 20

type IdempotencyMiddleware struct{ service *idempotency.Service }

func NewIdempotencyMiddleware(service *idempotency.Service) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{service: service}
}

func (m *IdempotencyMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get(idempotency.Header))
		if key == "" || !mutationMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		if len(key) > 255 {
			httputil.Error(w, apperror.NewBadRequest("idempotency key must not exceed 255 characters"))
			return
		}
		w.Header().Set(idempotency.Header, key)
		scope, ok := idempotencyScope(r)
		if !ok {
			httputil.Error(w, apperror.NewUnauthorized("authenticated idempotency scope required"))
			return
		}
		body, err := readIdempotentBody(r)
		_ = r.Body.Close()
		if err != nil {
			httputil.Error(w, err)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		digest := sha256.Sum256(body)
		request := idempotency.Request{
			Scope: scope, Key: key, Method: r.Method, Path: r.URL.RequestURI(),
			RequestHash: hex.EncodeToString(digest[:]),
		}

		claim, err := m.service.Claim(r.Context(), request)
		if err != nil {
			writeIdempotencyError(w, err)
			return
		}
		if claim.Response != nil {
			writeReplay(w, *claim.Response)
			return
		}

		captured := newBufferedResponseWriter()
		next.ServeHTTP(captured, r)
		response := idempotency.Response{
			Status: captured.status, Body: captured.body.Bytes(),
			ContentType: captured.header.Get("Content-Type"), Headers: replayHeaders(captured.header),
		}
		_ = m.service.Complete(r.Context(), request, claim.Lease, response)
		writeBuffered(w, captured)
	})
}

func mutationMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func idempotencyScope(r *http.Request) (string, bool) {
	if organizationID, ok := OrganizationIDFromContext(r.Context()); ok {
		return "organization:" + organizationID.String(), true
	}
	principal, ok := authn.PrincipalFromContext(r.Context())
	if !ok || !principal.IsValid() {
		return "", false
	}
	return string(principal.Subject.Type) + ":" + principal.Subject.ID.String(), true
}

func readIdempotentBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxIdempotentRequestBody+1))
	if err != nil {
		return nil, apperror.NewBadRequest("invalid request body")
	}
	if len(body) > maxIdempotentRequestBody {
		return nil, apperror.NewPayloadTooLarge("idempotent request body exceeds 1 MiB")
	}
	return body, nil
}

func writeIdempotencyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, idempotency.ErrKeyConflict):
		httputil.Error(w, apperror.NewConflict(err.Error()))
	case errors.Is(err, idempotency.ErrInProgress):
		w.Header().Set("Retry-After", "1")
		httputil.Error(w, apperror.NewConflict(err.Error()))
	default:
		httputil.Error(w, apperror.NewServiceUnavailable("idempotency service unavailable", err))
	}
}

func writeReplay(w http.ResponseWriter, response idempotency.Response) {
	for name, values := range response.Headers {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	if response.ContentType != "" {
		w.Header().Set("Content-Type", response.ContentType)
	}
	w.Header().Set("Idempotency-Replayed", "true")
	w.WriteHeader(response.Status)
	_, _ = w.Write(response.Body)
}

type bufferedResponseWriter struct {
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{header: make(http.Header), status: http.StatusOK}
}

func (w *bufferedResponseWriter) Header() http.Header { return w.header }

func (w *bufferedResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
}

func (w *bufferedResponseWriter) Write(body []byte) (int, error) {
	w.wroteHeader = true
	return w.body.Write(body)
}

func writeBuffered(w http.ResponseWriter, captured *bufferedResponseWriter) {
	for name, values := range captured.header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(captured.status)
	_, _ = w.Write(captured.body.Bytes())
}

func replayHeaders(header http.Header) map[string][]string {
	result := make(map[string][]string)
	for name, values := range header {
		switch http.CanonicalHeaderKey(name) {
		case "Connection", "Content-Length", "Content-Type", "Set-Cookie", "Transfer-Encoding":
			continue
		}
		result[name] = append([]string(nil), values...)
	}
	return result
}
