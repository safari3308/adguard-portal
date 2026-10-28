package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	corsMiddleware := CORS("*")

	// 1. Test OPTIONS preflight
	reqOptions := httptest.NewRequest(http.MethodOptions, "/api/devices", nil)
	recOptions := httptest.NewRecorder()

	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(recOptions, reqOptions)

	if recOptions.Code != http.StatusNoContent {
		t.Errorf("expected 204 No Content for OPTIONS, got %d", recOptions.Code)
	}
	if recOptions.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *")
	}

	// 2. Test GET request passing to next handler
	reqGet := httptest.NewRequest(http.MethodGet, "/api/devices", nil)
	recGet := httptest.NewRecorder()

	nextCalled := false
	handler2 := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	handler2.ServeHTTP(recGet, reqGet)

	if !nextCalled {
		t.Errorf("expected next handler to be called")
	}
	if recGet.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", recGet.Code)
	}
}
