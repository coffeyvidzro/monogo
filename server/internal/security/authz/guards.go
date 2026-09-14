package authz

// Guard evaluates whether a principal has a permission.
func Guard(principal Principal, permission Permission) bool {
	return Access{}.Allows(principal, permission)
}

// ScopedGuard evaluates whether a principal has a permission and, when the
// principal represents a scoped credential, the required scope.
func ScopedGuard(principal Principal, permission Permission, scope Scope) bool {
	return Access{}.AllowsScoped(principal, permission, scope)
}

// HasScope reports whether a credential contains a scope.
func HasScope(scopes []Scope, required Scope) bool {
	for _, scope := range scopes {
		if scope == required {
			return true
		}
	}
	return false
}
