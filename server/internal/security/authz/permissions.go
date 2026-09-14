package authz

// Permission identifies an action that a principal may perform.
type Permission string

const (
	PermissionOrganizationRead   Permission = "organization:read"
	PermissionOrganizationUpdate Permission = "organization:update"
	PermissionOrganizationDelete Permission = "organization:delete"

	PermissionMembersRead   Permission = "members:read"
	PermissionMembersAdd    Permission = "members:add"
	PermissionMembersUpdate Permission = "members:update"
	PermissionMembersRemove Permission = "members:remove"

	PermissionCredentialsRead   Permission = "credentials:read"
	PermissionCredentialsCreate Permission = "credentials:create"
	PermissionCredentialsRevoke Permission = "credentials:revoke"
)

func (p Permission) IsValid() bool {
	switch p {
	case PermissionOrganizationRead,
		PermissionOrganizationUpdate,
		PermissionOrganizationDelete,
		PermissionMembersRead,
		PermissionMembersAdd,
		PermissionMembersUpdate,
		PermissionMembersRemove,
		PermissionCredentialsRead,
		PermissionCredentialsCreate,
		PermissionCredentialsRevoke:
		return true
	default:
		return false
	}
}
