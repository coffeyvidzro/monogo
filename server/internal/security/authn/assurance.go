package authn

type AssuranceLevel uint8

const (
	AssuranceUnknown AssuranceLevel = iota
	AssurancePassword
	AssuranceOTP
	AssuranceMFA
)

func (a AssuranceLevel) String() string {
	switch a {
	case AssurancePassword:
		return "password"
	case AssuranceOTP:
		return "otp"
	case AssuranceMFA:
		return "mfa"
	default:
		return "unknown"
	}
}
