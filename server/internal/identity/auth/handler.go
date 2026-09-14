package auth

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/leamout/leamout/internal/identity/session"
	"github.com/leamout/leamout/internal/security/authn"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/helper"
	"github.com/leamout/leamout/pkg/httputil"
)

type Handler struct {
	service        *Service
	sessionService *session.Service
}

func NewHandler(
	service *Service,
	sessionService *session.Service,
) *Handler {
	return &Handler{
		service:        service,
		sessionService: sessionService,
	}
}

func (h *Handler) Start(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := helper.DecodeJSON[startRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	email, err := validateStartRequest(req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	transaction, err := h.service.Start(
		r.Context(),
		email,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(
		w,
		startResponse{
			TransactionID: transaction.ID,
			Methods:       authenticationMethods(),
		},
	)
}

func (h *Handler) LoginWithPassword(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := helper.DecodeJSON[passwordLoginRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	transactionID, err := validatePasswordLoginRequest(req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	user, err := h.service.LoginWithPassword(
		r.Context(),
		transactionID,
		req.Password,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	h.createSession(w, r, user.ID, authn.AssurancePassword)
}

func (h *Handler) SendOTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := helper.DecodeJSON[sendOTPRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	transactionID, err := validateSendOTPRequest(req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if _, err := h.service.SendOTP(
		r.Context(),
		transactionID,
	); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(
		w,
		sendOTPResponse{
			TransactionID: transactionID,
		},
	)
}

func (h *Handler) VerifyOTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := helper.DecodeJSON[verifyOTPRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	transactionID, code, err := validateVerifyOTPRequest(req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	user, err := h.service.VerifyOTP(
		r.Context(),
		transactionID,
		code,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	h.createSession(w, r, user.ID, authn.AssuranceOTP)
}

func (h *Handler) SetPassword(
	w http.ResponseWriter,
	r *http.Request,
) {
	req, err := helper.DecodeJSON[setPasswordRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := validateSetPasswordRequest(req); err != nil {
		httputil.Error(w, err)
		return
	}

	userID, ok := authn.UserIDFromContext(r.Context())
	if !ok {
		httputil.Error(
			w,
			apperror.NewUnauthorized("authentication required"),
		)
		return
	}

	user, err := h.service.SetPassword(
		r.Context(),
		userID,
		req.Password,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(
		w,
		authenticationResponse{
			UserID: user.ID,
		},
	)
}

func (h *Handler) createSession(
	w http.ResponseWriter,
	r *http.Request,
	userID uuid.UUID,
	assurance authn.AssuranceLevel,
) {
	token, sess, err := h.sessionService.CreateWithAssurance(
		r.Context(),
		userID,
		helper.ClientIP(r),
		helper.UserAgent(r),
		assurance,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	session.SetCookie(
		w,
		token,
		sess.ExpiresAt.Time,
	)

	httputil.OK(
		w,
		authenticationResponse{
			UserID:        userID,
			SessionExpiry: sess.ExpiresAt.Time.Format(http.TimeFormat),
		},
	)
}

func authenticationMethods() []string {
	return []string{"password", "otp"}
}
