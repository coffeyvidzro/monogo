package authz

// Allows reports whether a role grants a permission.
func Allows(role Role, permission Permission) bool {
	switch role {
	case RoleOwner:
		return permission.IsValid()
	case RoleAdmin:
		switch permission {
		case PermissionOrganizationRead, PermissionOrganizationUpdate,
			PermissionMembersRead, PermissionMembersAdd,
			PermissionMembersUpdate, PermissionMembersRemove,
			PermissionCredentialsRead, PermissionCredentialsCreate,
			PermissionCredentialsRevoke:
			return true
		}
	case RoleMember:
		switch permission {
		case PermissionOrganizationRead, PermissionMembersRead:
			return true
		}
	}
	return false
}
