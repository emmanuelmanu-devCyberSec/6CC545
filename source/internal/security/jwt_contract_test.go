//go:build acceptance

package security

import (
	"errors"
	jwt "github.com/golang-jwt/jwt/v5"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var testTime = time.Unix(1800000000, 0)
var testKey = []byte("local-test-only-32-byte-randomness-not-for-deployment")

func validator() TokenValidator {
	return TokenValidator{Key: testKey, Issuer: "cw2-lab", Audience: "cw2-api", Now: func() time.Time { return testTime }}
}
func claims(role string) Claims {
	return Claims{Role: role, RegisteredClaims: jwt.RegisteredClaims{Subject: "student", Issuer: "cw2-lab", Audience: jwt.ClaimStrings{"cw2-api"}, ExpiresAt: jwt.NewNumericDate(testTime.Add(time.Minute)), NotBefore: jwt.NewNumericDate(testTime.Add(-time.Minute))}}
}
func signed(t *testing.T, c Claims, key []byte, method jwt.SigningMethod) string {
	t.Helper()
	s, e := jwt.NewWithClaims(method, c).SignedString(key)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func TestValidRoles(t *testing.T) {
	v := validator()
	for _, role := range []string{"reader", "writer"} {
		c, e := v.Validate(signed(t, claims(role), testKey, jwt.SigningMethodHS256), "reader")
		if e != nil || c.Subject != "student" {
			t.Fatalf("valid role %s: %v", role, e)
		}
	}
}
func TestJWTRejections(t *testing.T) {
	v := validator()
	base := claims("reader")
	expired := base
	expired.ExpiresAt = jwt.NewNumericDate(testTime.Add(-time.Second))
	noexp := base
	noexp.ExpiresAt = nil
	future := base
	future.NotBefore = jwt.NewNumericDate(testTime.Add(time.Minute))
	wrongiss := base
	wrongiss.Issuer = "other"
	wrongaud := base
	wrongaud.Audience = jwt.ClaimStrings{"other"}
	nosub := base
	nosub.Subject = ""
	badrole := claims("administrator")
	bad := []string{"", "not-a-jwt", signed(t, expired, testKey, jwt.SigningMethodHS256), signed(t, noexp, testKey, jwt.SigningMethodHS256), signed(t, future, testKey, jwt.SigningMethodHS256), signed(t, wrongiss, testKey, jwt.SigningMethodHS256), signed(t, wrongaud, testKey, jwt.SigningMethodHS256), signed(t, nosub, testKey, jwt.SigningMethodHS256), signed(t, badrole, testKey, jwt.SigningMethodHS256), signed(t, base, []byte("wrong-signature-key"), jwt.SigningMethodHS256), signed(t, base, testKey, jwt.SigningMethodHS384)}
	for i, raw := range bad {
		if _, e := v.Validate(raw, "reader"); e == nil {
			t.Errorf("invalid credential case %d accepted", i)
		}
	}
	if _, e := v.Validate(signed(t, base, testKey, jwt.SigningMethodHS256), "writer"); !errors.Is(e, ErrForbidden) {
		t.Fatalf("reader write permission: %v", e)
	}
}
func TestMiddleware(t *testing.T) {
	called := false
	h := Require(validator(), "writer", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(204) }))
	for _, tc := range []struct {
		role string
		want int
	}{{"reader", 403}, {"writer", 204}} {
		called = false
		r := httptest.NewRequest("POST", "/v1/files", strings.NewReader("x"))
		r.Header.Set("Authorization", "Bearer "+signed(t, claims(tc.role), testKey, jwt.SigningMethodHS256))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want || called != (tc.want == 204) {
			t.Fatalf("%s got %d called=%v", tc.role, w.Code, called)
		}
	}
}

func TestRequiredNBFAndConfiguration(t *testing.T) {
	c := claims("reader")
	c.NotBefore = nil
	if _, e := validator().Validate(signed(t, c, testKey, jwt.SigningMethodHS256), "reader"); e == nil {
		t.Fatal("missing nbf accepted")
	}
	v := validator()
	v.Key = []byte("short")
	if _, e := v.Validate(signed(t, claims("reader"), testKey, jwt.SigningMethodHS256), "reader"); e == nil {
		t.Fatal("short key configuration accepted")
	}
}
func TestVerifiedClaimsReachHandler(t *testing.T) {
	h := Require(validator(), "reader", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := ClaimsFromContext(r.Context())
		if !ok || c.Subject != "student" {
			t.Error("verified identity absent from request context")
		}
		w.WriteHeader(204)
	}))
	r := httptest.NewRequest("GET", "/v1/files", nil)
	r.Header.Set("Authorization", "Bearer "+signed(t, claims("reader"), testKey, jwt.SigningMethodHS256))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatalf("valid request %d", w.Code)
	}
}
