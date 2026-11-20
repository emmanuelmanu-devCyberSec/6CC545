//go:build acceptance

package security

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	jwt "github.com/golang-jwt/jwt/v5"
)

func TestRejectionLogIncludesNodeOperationAndReason(t *testing.T) {
	v := validator()
	v.NodeID = "node-a"

	handler := Require(
		v,
		"writer",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)

	// Capture log output without writing test logs to the console.
	var output strings.Builder
	previousOutput := log.Writer()
	previousFlags := log.Flags()
	log.SetOutput(&output)
	log.SetFlags(0)

	defer func() {
		log.SetOutput(previousOutput)
		log.SetFlags(previousFlags)
	}()

	request := httptest.NewRequest(http.MethodPost, "/v1/files", nil)
	request.Header.Set(
		"Authorization",
		"Bearer "+signed(t, claims("reader"), testKey, jwtSigningMethodHS256()),
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, response.Code)
	}

	got := output.String()

	for _, expected := range []string{
		"node=node-a",
		"operation=POST /v1/files",
		"reason=insufficient_role",
	} {
		if !strings.Contains(got, expected) {
			t.Errorf("rejection log missing %q: %s", expected, got)
		}
	}

	if strings.Contains(got, "Bearer ") {
		t.Fatal("rejection log must not contain bearer credentials")
	}
}

// Keep the test's signing method consistent with the JWT contract.
func jwtSigningMethodHS256() jwt.SigningMethod {
	return jwt.SigningMethodHS256
}
