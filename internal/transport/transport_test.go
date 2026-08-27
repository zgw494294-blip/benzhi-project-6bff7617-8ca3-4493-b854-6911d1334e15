package transport

import (
	"fieldlingua/internal/application"
	"fieldlingua/internal/persistence"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	s := New(application.New(persistence.New(t.TempDir())))
	r := httptest.NewRecorder()
	q := httptest.NewRequest("GET", "/healthz", nil)
	s.Handler().ServeHTTP(r, q)
	if r.Code != 200 {
		t.Fatal(r.Code)
	}
}
