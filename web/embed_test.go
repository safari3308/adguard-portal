package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDistHandler(t *testing.T) {
	handler := DistHandler()

	// 1. Root path -> should return index.html (status 200)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for root path, got %d", rec.Code)
	}

	// 2. SPA client-side route fallback -> should also return index.html (status 200)
	req2 := httptest.NewRequest(http.MethodGet, "/some/spa/route", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200 for SPA fallback, got %d", rec2.Code)
	}
}
