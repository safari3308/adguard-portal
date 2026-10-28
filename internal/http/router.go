package http

import (
	"net/http"

	"github.com/safari3308/adguard-portal/internal/auth"
	"github.com/safari3308/adguard-portal/internal/http/handler"
	"github.com/safari3308/adguard-portal/internal/http/middleware"
	"github.com/safari3308/adguard-portal/web"
)

type RouterConfig struct {
	UserHandler    *handler.UserHandler
	DeviceHandler  *handler.DeviceHandler
	ServiceHandler *handler.ServiceHandler
	AuthValidator  *auth.Validator
	CORSOrigin     string
}

func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Public config (for SPA OIDC init)
	mux.HandleFunc("GET /api/config", cfg.ServiceHandler.PublicConfig)

	// Authenticated sub-router / handlers
	authMiddleware := cfg.AuthValidator.Middleware

	mux.Handle("GET /api/me", authMiddleware(http.HandlerFunc(cfg.UserHandler.Me)))
	mux.Handle("GET /api/devices", authMiddleware(http.HandlerFunc(cfg.DeviceHandler.List)))
	mux.Handle("POST /api/devices", authMiddleware(http.HandlerFunc(cfg.DeviceHandler.Create)))
	mux.Handle("GET /api/devices/{id}", authMiddleware(http.HandlerFunc(cfg.DeviceHandler.Get)))
	mux.Handle("PATCH /api/devices/{id}", authMiddleware(http.HandlerFunc(cfg.DeviceHandler.Update)))
	mux.Handle("DELETE /api/devices/{id}", authMiddleware(http.HandlerFunc(cfg.DeviceHandler.Delete)))
	mux.Handle("POST /api/devices/{id}/ticket", authMiddleware(http.HandlerFunc(cfg.DeviceHandler.CreateTicket)))
	mux.Handle("GET /api/devices/{id}/mobileconfig", authMiddleware(http.HandlerFunc(cfg.DeviceHandler.DownloadMobileConfig)))
	mux.Handle("GET /api/services", authMiddleware(http.HandlerFunc(cfg.ServiceHandler.ListServices)))

	// Static SPA handler for all other paths
	mux.Handle("/", web.DistHandler())

	// Apply global middleware (CORS)
	return middleware.CORS(cfg.CORSOrigin)(mux)
}
