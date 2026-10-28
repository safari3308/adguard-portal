package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/safari3308/adguard-portal/internal/auth"
	"github.com/safari3308/adguard-portal/internal/domain"
	"github.com/safari3308/adguard-portal/internal/http/handler"
	"github.com/safari3308/adguard-portal/internal/repository/sqlite"
	"github.com/safari3308/adguard-portal/internal/service"
)

func TestNewRouter(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("sqlite.Open failed: %v", err)
	}
	defer db.Close()

	userRepo := sqlite.NewUserRepo(db)
	deviceRepo := sqlite.NewDeviceRepo(db)

	testUser := &domain.User{ID: "dev-user-router", Email: "dev@router.local", Name: "Dev Router"}
	_ = userRepo.Upsert(context.Background(), testUser)

	devService := service.NewDeviceService(deviceRepo, nil, 10, "dns.router.local")
	devHandler := handler.NewDeviceHandler(devService)
	userHandler := handler.NewUserHandler()
	serviceHandler := handler.NewServiceHandler(nil, handler.ConfigResponse{
		DOHDomain: "dns.router.local",
		DevMode:   true,
	})

	authValidator := auth.NewValidator("https://issuer.local", "", "client-id", true, userRepo)

	router := NewRouter(RouterConfig{
		UserHandler:    userHandler,
		DeviceHandler:  devHandler,
		ServiceHandler: serviceHandler,
		AuthValidator:  authValidator,
		CORSOrigin:     "*",
	})

	// 1. Health check
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	router.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Errorf("expected 200 for /health, got %d", recHealth.Code)
	}

	// 2. Public config
	reqConfig := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	recConfig := httptest.NewRecorder()
	router.ServeHTTP(recConfig, reqConfig)
	if recConfig.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/config, got %d", recConfig.Code)
	}

	// 3. Authenticated route with Dev Mode Bearer token
	reqMe := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	reqMe.Header.Set("Authorization", "Bearer dev-user-router")
	recMe := httptest.NewRecorder()
	router.ServeHTTP(recMe, reqMe)
	if recMe.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/me with dev token, got %d", recMe.Code)
	}

	// 4. Authenticated route without token -> 401
	reqMeNoAuth := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	recMeNoAuth := httptest.NewRecorder()
	router.ServeHTTP(recMeNoAuth, reqMeNoAuth)
	if recMeNoAuth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for /api/me without auth, got %d", recMeNoAuth.Code)
	}

	// 5. Static SPA route
	reqSPA := httptest.NewRequest(http.MethodGet, "/", nil)
	recSPA := httptest.NewRecorder()
	router.ServeHTTP(recSPA, reqSPA)
	if recSPA.Code != http.StatusOK {
		t.Errorf("expected 200 for root SPA route, got %d", recSPA.Code)
	}
}
