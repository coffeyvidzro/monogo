package authn

import "github.com/google/uuid"

type SubjectType string

const (
	SubjectUser              SubjectType = "user"
	SubjectOrganizationToken SubjectType = "organization_token"
)

type Subject struct {
	ID   uuid.UUID
	Type SubjectType
}

type Principal struct {
	Subject        Subject
	Credential     Credential
	OrganizationID uuid.UUID
	Scopes         []string
	Assurance      AssuranceLevel
}

func (p Principal) IsValid() bool {
	return p.Subject.ID != uuid.Nil && p.Subject.Type != "" && p.Credential.Type != ""
}

func (p Principal) UserID() (uuid.UUID, bool) {
	if p.Subject.Type != SubjectUser || p.Subject.ID == uuid.Nil {
		return uuid.Nil, false
	}

	return p.Subject.ID, true
}
