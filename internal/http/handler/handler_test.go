package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/safari3308/adguard-portal/internal/auth"
	"github.com/safari3308/adguard-portal/internal/domain"
	"github.com/safari3308/adguard-portal/internal/repository/sqlite"
	"github.com/safari3308/adguard-portal/internal/service"
)

type mockAGHClient struct {
	services []domain.BlockedService
	err      error
}

func (m *mockAGHClient) SyncDevice(ctx context.Context, device *domain.Device) error { return nil }
func (m *mockAGHClient) DeleteDevice(ctx context.Context, clientToken, deviceName string) error {
	return nil
}
func (m *mockAGHClient) EnsureAllowedClient(ctx context.Context, clientToken string) error {
	return nil
}
func (m *mockAGHClient) RemoveAllowedClient(ctx context.Context, clientToken string) error {
	return nil
}
func (m *mockAGHClient) GetBlockedServices(ctx context.Context) ([]domain.BlockedService, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.services, nil
}

func TestUserHandler(t *testing.T) {
	h := NewUserHandler()

	// 1. Unauthorized
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	h.Me(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}

	// 2. Authorized
	testUser := &domain.User{ID: "usr-1", Email: "user@example.com", Name: "User One"}
	reqAuth := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	reqAuth = reqAuth.WithContext(auth.WithUser(reqAuth.Context(), testUser))
	recAuth := httptest.NewRecorder()
	h.Me(recAuth, reqAuth)
	if recAuth.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recAuth.Code)
	}
}

func TestServiceHandler(t *testing.T) {
	cfg := ConfigResponse{
		DOHDomain: "dns.example.com",
		DevMode:   true,
	}

	// 1. Nil AGH
	hNil := NewServiceHandler(nil, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	rec := httptest.NewRecorder()
	hNil.ListServices(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 on nil AGH, got %d", rec.Code)
	}

	// 2. AGH success
	mockAGH := &mockAGHClient{
		services: []domain.BlockedService{{ID: "youtube", Name: "YouTube"}},
	}
	h := NewServiceHandler(mockAGH, cfg)
	rec2 := httptest.NewRecorder()
	h.ListServices(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec2.Code)
	}

	// 3. AGH error
	mockErr := &mockAGHClient{err: errors.New("agh timeout")}
	hErr := NewServiceHandler(mockErr, cfg)
	rec3 := httptest.NewRecorder()
	hErr.ListServices(rec3, req)
	if rec3.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on AGH error, got %d", rec3.Code)
	}

	// 4. PublicConfig
	recCfg := httptest.NewRecorder()
	h.PublicConfig(recCfg, req)
	if recCfg.Code != http.StatusOK {
		t.Errorf("expected 200 on PublicConfig, got %d", recCfg.Code)
	}
}

func TestDeviceHandler(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("sqlite.Open failed: %v", err)
	}
	defer db.Close()

	userRepo := sqlite.NewUserRepo(db)
	deviceRepo := sqlite.NewDeviceRepo(db)

	testUser := &domain.User{ID: "usr-device", Email: "dev@example.com", Name: "Dev"}
	_ = userRepo.Upsert(context.Background(), testUser)

	deviceService := service.NewDeviceService(deviceRepo, nil, 1, "dns.example.com")
	h := NewDeviceHandler(deviceService)

	withAuth := func(req *http.Request) *http.Request {
		return req.WithContext(auth.WithUser(req.Context(), testUser))
	}

	// 1. List initially empty
	reqList := withAuth(httptest.NewRequest(http.MethodGet, "/api/devices", nil))
	recList := httptest.NewRecorder()
	h.List(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Errorf("expected 200 for List, got %d", recList.Code)
	}

	// 2. Create device - invalid json
	reqBadJSON := withAuth(httptest.NewRequest(http.MethodPost, "/api/devices", bytes.NewReader([]byte("{invalid"))))
	recBadJSON := httptest.NewRecorder()
	h.Create(recBadJSON, reqBadJSON)
	if recBadJSON.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json body, got %d", recBadJSON.Code)
	}

	// 3. Create device - success
	body, _ := json.Marshal(domain.CreateDeviceInput{Name: "Work Laptop"})
	reqCreate := withAuth(httptest.NewRequest(http.MethodPost, "/api/devices", bytes.NewReader(body)))
	recCreate := httptest.NewRecorder()
	h.Create(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 for Create, got %d", recCreate.Code)
	}

	var created domain.Device
	_ = json.NewDecoder(recCreate.Body).Decode(&created)

	// 4. Create device - limit reached (limit is 1)
	reqLimit := withAuth(httptest.NewRequest(http.MethodPost, "/api/devices", bytes.NewReader(body)))
	recLimit := httptest.NewRecorder()
	h.Create(recLimit, reqLimit)
	if recLimit.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict when limit reached, got %d", recLimit.Code)
	}

	// 5. Get device - success
	reqGet := withAuth(httptest.NewRequest(http.MethodGet, "/api/devices/"+created.ID, nil))
	reqGet.SetPathValue("id", created.ID)
	recGet := httptest.NewRecorder()
	h.Get(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Errorf("expected 200 for Get, got %d", recGet.Code)
	}

	// Get device - not found
	reqGet404 := withAuth(httptest.NewRequest(http.MethodGet, "/api/devices/fake", nil))
	reqGet404.SetPathValue("id", "fake")
	recGet404 := httptest.NewRecorder()
	h.Get(recGet404, reqGet404)
	if recGet404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for Get fake, got %d", recGet404.Code)
	}

	// 6. Update device - invalid JSON
	reqUpBad := withAuth(httptest.NewRequest(http.MethodPatch, "/api/devices/"+created.ID, bytes.NewReader([]byte("{invalid"))))
	reqUpBad.SetPathValue("id", created.ID)
	recUpBad := httptest.NewRecorder()
	h.Update(recUpBad, reqUpBad)
	if recUpBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for Update invalid JSON, got %d", recUpBad.Code)
	}

	// Update device - success
	newName := "Home Desktop"
	upBody, _ := json.Marshal(domain.UpdateDeviceInput{Name: &newName})
	reqUp := withAuth(httptest.NewRequest(http.MethodPatch, "/api/devices/"+created.ID, bytes.NewReader(upBody)))
	reqUp.SetPathValue("id", created.ID)
	recUp := httptest.NewRecorder()
	h.Update(recUp, reqUp)
	if recUp.Code != http.StatusOK {
		t.Errorf("expected 200 for Update, got %d", recUp.Code)
	}

	// Update device - invalid domain rule
	badDomain := []string{"invalid domain!!"}
	upBadRule, _ := json.Marshal(domain.UpdateDeviceInput{CustomBlockedDomains: &badDomain})
	reqUpRule := withAuth(httptest.NewRequest(http.MethodPatch, "/api/devices/"+created.ID, bytes.NewReader(upBadRule)))
	reqUpRule.SetPathValue("id", created.ID)
	recUpRule := httptest.NewRecorder()
	h.Update(recUpRule, reqUpRule)
	if recUpRule.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid domain rule, got %d", recUpRule.Code)
	}

	// Update device - not found
	reqUp404 := withAuth(httptest.NewRequest(http.MethodPatch, "/api/devices/fake", bytes.NewReader(upBody)))
	reqUp404.SetPathValue("id", "fake")
	recUp404 := httptest.NewRecorder()
	h.Update(recUp404, reqUp404)
	if recUp404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for Update fake, got %d", recUp404.Code)
	}

	// 7. DownloadMobileConfig
	reqMobile := withAuth(httptest.NewRequest(http.MethodGet, "/api/devices/"+created.ID+"/mobileconfig", nil))
	reqMobile.SetPathValue("id", created.ID)
	recMobile := httptest.NewRecorder()
	h.DownloadMobileConfig(recMobile, reqMobile)
	if recMobile.Code != http.StatusOK {
		t.Errorf("expected 200 for MobileConfig, got %d", recMobile.Code)
	}
	if recMobile.Header().Get("Content-Type") != "application/x-apple-aspen-config" {
		t.Errorf("unexpected content type: %s", recMobile.Header().Get("Content-Type"))
	}

	// DownloadMobileConfig - not found
	reqMobile404 := withAuth(httptest.NewRequest(http.MethodGet, "/api/devices/fake/mobileconfig", nil))
	reqMobile404.SetPathValue("id", "fake")
	recMobile404 := httptest.NewRecorder()
	h.DownloadMobileConfig(recMobile404, reqMobile404)
	if recMobile404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for MobileConfig fake, got %d", recMobile404.Code)
	}

	// 8. Delete device
	reqDel := withAuth(httptest.NewRequest(http.MethodDelete, "/api/devices/"+created.ID, nil))
	reqDel.SetPathValue("id", created.ID)
	recDel := httptest.NewRecorder()
	h.Delete(recDel, reqDel)
	if recDel.Code != http.StatusNoContent {
		t.Errorf("expected 204 for Delete, got %d", recDel.Code)
	}

	// Delete device - not found
	reqDel404 := withAuth(httptest.NewRequest(http.MethodDelete, "/api/devices/fake", nil))
	reqDel404.SetPathValue("id", "fake")
	recDel404 := httptest.NewRecorder()
	h.Delete(recDel404, reqDel404)
	if recDel404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for Delete fake, got %d", recDel404.Code)
	}
}
