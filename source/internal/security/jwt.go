package security

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

// Claims contains the verified identity and role from a JWT.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// TokenValidator holds the configuration used to validate JWTs.
type TokenValidator struct {
	Key              []byte
	Issuer, Audience string
	NodeID           string
	Now              func() time.Time
}

// Validate checks the token signature, registered claims and required role.
func (v TokenValidator) Validate(raw, requiredRole string) (Claims, error) {
	if strings.TrimSpace(raw) == "" || len(raw) > 8*1024 {
		return Claims{}, fmt.Errorf("%w: token is empty or too large", ErrUnauthenticated)
	}

	if len(v.Key) < 32 {
		return Claims{}, fmt.Errorf("%w: signing key must be at least 32 bytes", ErrUnauthenticated)
	}

	if strings.TrimSpace(v.Issuer) == "" || strings.TrimSpace(v.Audience) == "" {
		return Claims{}, fmt.Errorf("%w: issuer and audience are required", ErrUnauthenticated)
	}

	now := v.Now
	if now == nil {
		now = time.Now
	}

	var claims Claims

	token, err := jwt.ParseWithClaims(
		raw,
		&claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return v.Key, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(v.Issuer),
		jwt.WithAudience(v.Audience),
		jwt.WithExpirationRequired(),
		jwt.WithNotBeforeRequired(),
		jwt.WithTimeFunc(now),
	)
	if err != nil || token == nil || !token.Valid {
		return Claims{}, fmt.Errorf("%w: token validation failed", ErrUnauthenticated)
	}

	if strings.TrimSpace(claims.Subject) == "" {
		return Claims{}, fmt.Errorf("%w: token subject is missing", ErrUnauthenticated)
	}

	if claims.Role != "reader" && claims.Role != "writer" {
		return Claims{}, fmt.Errorf("%w: token role is not supported", ErrUnauthenticated)
	}

	if requiredRole != "reader" && requiredRole != "writer" {
		return Claims{}, ErrForbidden
	}

	if requiredRole == "writer" && claims.Role != "writer" {
		return Claims{}, ErrForbidden
	}

	return claims, nil
}

// Require validates the Bearer token before passing the request to the handler.
// Rejection logs contain the node, operation and reason category, never credentials.
func Require(v TokenValidator, requiredRole string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reject := func(status int, reason string) {
			log.Printf(
				"security_rejection node=%s operation=%s %s reason=%s",
				v.NodeID,
				r.Method,
				r.URL.Path,
				reason,
			)

			if status == http.StatusUnauthorized {
				w.Header().Set("WWW-Authenticate", "Bearer")
			}

			http.Error(w, http.StatusText(status), status)
		}

		headers := r.Header.Values("Authorization")
		if len(headers) != 1 {
			reject(http.StatusUnauthorized, "missing_or_multiple_authorization_headers")
			return
		}

		parts := strings.Fields(headers[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			reject(http.StatusUnauthorized, "invalid_authorization_header")
			return
		}

		claims, err := v.Validate(parts[1], requiredRole)
		if err != nil {
			if errors.Is(err, ErrForbidden) {
				reject(http.StatusForbidden, "insufficient_role")
				return
			}

			reject(http.StatusUnauthorized, "invalid_token")
			return
		}

		ctx := context.WithValue(r.Context(), privateContextKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
