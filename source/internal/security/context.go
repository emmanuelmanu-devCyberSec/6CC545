package security

import "context"

// privateContextKey prevents another package from colliding with this key.
type privateContextKey struct{}

// ClaimsFromContext is supplied infrastructure. Require must install verified
// Claims using context.WithValue(r.Context(), privateContextKey{}, claims).
// Never store or return an unvalidated token's claims here.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(privateContextKey{}).(Claims)
	return claims, ok
}
