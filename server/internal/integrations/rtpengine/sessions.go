package rtpengine

import (
	"fmt"
	"strings"
	"time"
)

// NewSession creates a validated RTPEngine session.
func NewSession(callID, fromTag, branch string) (Session, error) {
	session := Session{
		CallID:    strings.TrimSpace(callID),
		FromTag:   strings.TrimSpace(fromTag),
		Branch:    strings.TrimSpace(branch),
		CreatedAt: time.Now().UTC(),
	}
	if err := session.Validate(); err != nil {
		return Session{}, err
	}
	return session, nil
}

// WithToTag returns a copy of the session with the normalized to-tag.
func (s Session) WithToTag(toTag string) Session {
	s.ToTag = strings.TrimSpace(toTag)
	return s
}

// String returns a stable human-readable session identifier.
func (s Session) String() string {
	if s.ToTag == "" {
		return fmt.Sprintf("%s/%s/%s", s.CallID, s.FromTag, s.Branch)
	}
	return fmt.Sprintf("%s/%s/%s/%s", s.CallID, s.FromTag, s.ToTag, s.Branch)
}

func sessionParams(session Session, sdp string, flags []string) map[string]any {
	params := map[string]any{
		"call-id":    session.CallID,
		"from-tag":   session.FromTag,
		"via-branch": session.Branch,
	}
	if session.ToTag != "" {
		params["to-tag"] = session.ToTag
	}
	if sdp != "" {
		params["sdp"] = sdp
	}
	if len(flags) > 0 {
		params["flags"] = append([]string(nil), flags...)
	}
	return params
}
