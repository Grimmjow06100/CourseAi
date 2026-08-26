package contract

import (
	"context"
	"strings"
)

type principalContextKey struct{}

// Principal is the authenticated identity made available to application use cases.
type Principal struct {
	UserID    string
	SessionID string
}

func (p Principal) Validate() error {
	if strings.TrimSpace(p.UserID) == "" {
		return ErrUnauthenticated
	}
	return nil
}

func ContextWithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok && principal.Validate() == nil
}
