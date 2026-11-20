package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScaffoldFailsClosed(t *testing.T) {
	called := false
	h := Require(TokenValidator{}, "reader", http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if called || w.Code != 401 {
		t.Fatal("missing credentials must be blocked")
	}
}
