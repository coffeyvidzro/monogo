package checkout

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func validateCreateRequest(req *CreateRequest) error {
	if req.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: organization id is required", ErrInvalidInput)
	}

	req.Purpose = strings.ToLower(strings.TrimSpace(req.Purpose))
	switch req.Purpose {
	case PurposeSubscription:
		if req.SubscriptionID == nil || *req.SubscriptionID == uuid.Nil {
			return fmt.Errorf("%w: subscription id is required", ErrInvalidInput)
		}
		if req.AmountMicros != 0 {
			return fmt.Errorf("%w: subscription amount is derived by checkout", ErrInvalidInput)
		}
	case PurposeWalletTopup:
		if req.SubscriptionID != nil {
			return fmt.Errorf("%w: wallet top-up cannot include subscription id", ErrInvalidInput)
		}
		if req.AmountMicros <= 0 {
			return fmt.Errorf("%w: amount must be greater than zero", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: purpose is invalid", ErrInvalidInput)
	}

	return nil
}

func validateConfirmRequest(req *ConfirmRequest) error {
	if req.OrganizationID == uuid.Nil || req.CheckoutID == uuid.Nil {
		return fmt.Errorf("%w: organization and checkout ids are required", ErrInvalidInput)
	}

	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.PaymentMethod = strings.ToLower(strings.TrimSpace(req.PaymentMethod))

	switch {
	case req.Provider == ProviderStripe && req.PaymentMethod == PaymentMethodCard:
		return nil
	case req.Provider == ProviderPaystack && req.PaymentMethod == PaymentMethodMobileMoney:
		return nil
	default:
		return fmt.Errorf("%w: provider and payment method are incompatible", ErrInvalidInput)
	}
}

func validateContinueRequest(req *ContinueRequest) error {
	if req.OrganizationID == uuid.Nil || req.CheckoutID == uuid.Nil {
		return fmt.Errorf("%w: organization and checkout ids are required", ErrInvalidInput)
	}

	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	switch req.Action {
	case ActionAuthorizeMobileMoney:
		if req.Phone != nil || req.OTP != nil {
			return fmt.Errorf("%w: authorization does not accept continuation data", ErrInvalidInput)
		}
	case ActionSubmitPhone:
		if req.Phone == nil {
			return fmt.Errorf("%w: phone is required", ErrInvalidInput)
		}
		phone := strings.TrimSpace(*req.Phone)
		if phone == "" {
			return fmt.Errorf("%w: phone is required", ErrInvalidInput)
		}
		req.Phone = &phone
		if req.OTP != nil {
			return fmt.Errorf("%w: otp is not accepted for phone continuation", ErrInvalidInput)
		}
	case ActionSubmitOTP:
		if req.OTP == nil {
			return fmt.Errorf("%w: otp is required", ErrInvalidInput)
		}
		otp := strings.TrimSpace(*req.OTP)
		if len(otp) < 4 || len(otp) > 10 {
			return fmt.Errorf("%w: otp must be between 4 and 10 characters", ErrInvalidInput)
		}
		req.OTP = &otp
		if req.Phone != nil {
			return fmt.Errorf("%w: phone is not accepted for otp continuation", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: checkout action cannot be continued", ErrInvalidInput)
	}

	return nil
}
