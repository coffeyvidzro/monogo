package session

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/coffeyvidzro/monogo/internal/security/authn"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/httputil"
)

type Handler struct {
	service      *Service
	development  bool
	cookieDomain string
}

func NewHandler(
	service *Service,
	development bool,
	cookieDomain string,
) *Handler {
	return &Handler{
		service:      service,
		development:  development,
		cookieDomain: strings.TrimSpace(cookieDomain),
	}
}

func (h *Handler) userID(
	r *http.Request,
) (uuid.UUID, error) {
	userID, ok := authn.UserIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, apperror.NewUnauthorized("authentication required")
	}

	return userID, nil
}

func (h *Handler) sessionID(
	r *http.Request,
) (uuid.UUID, error) {
	principal, ok := authn.PrincipalFromContext(r.Context())
	if !ok {
		return uuid.Nil, apperror.NewUnauthorized("session authentication required")
	}

	if principal.Credential.Type != authn.CredentialSession {
		return uuid.Nil, apperror.NewUnauthorized("session authentication required")
	}

	if principal.Credential.ID == uuid.Nil {
		return uuid.Nil, apperror.NewUnauthorized("session authentication required")
	}

	return principal.Credential.ID, nil
}

func (h *Handler) List(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, err := h.userID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	sessions, err := h.service.List(r.Context(), userID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	items := make([]Response, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, toResponse(session))
	}

	httputil.OK(w, map[string]any{"sessions": items})
}

func (h *Handler) Revoke(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, err := h.userID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid session id"))
		return
	}

	if err := h.service.Revoke(r.Context(), sessionID, userID); err != nil {
		httputil.Error(w, err)
		return
	}

	currentSessionID, err := h.sessionID(r)
	if err == nil && currentSessionID == sessionID {
		ClearCookie(w, h.development, h.cookieDomain)
	}

	httputil.OK(w, map[string]string{"message": "session revoked"})
}

func (h *Handler) RevokeAll(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, err := h.userID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.RevokeAll(r.Context(), userID); err != nil {
		httputil.Error(w, err)
		return
	}

	ClearCookie(w, h.development, h.cookieDomain)

	httputil.OK(w, map[string]string{"message": "logged out from all sessions"})
}

func SetCookie(
	w http.ResponseWriter,
	token string,
	expiresAt time.Time,
	development bool,
	cookieDomain string,
) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authn.SessionCookieName,
		Value:    token,
		Path:     "/",
		Domain:   cookieDomainValue(development, cookieDomain),
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   !development,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearCookie(
	w http.ResponseWriter,
	development bool,
	cookieDomain string,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     authn.SessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   cookieDomainValue(development, cookieDomain),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !development,
		SameSite: http.SameSiteLaxMode,
	})
}

func cookieDomainValue(development bool, domain string) string {
	if development {
		return ""
	}
	return strings.TrimSpace(domain)
}
