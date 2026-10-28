package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/safari3308/adguard-portal/internal/agh"
	"github.com/safari3308/adguard-portal/internal/auth"
	"github.com/safari3308/adguard-portal/internal/config"
	appHTTP "github.com/safari3308/adguard-portal/internal/http"
	"github.com/safari3308/adguard-portal/internal/http/handler"
	"github.com/safari3308/adguard-portal/internal/repository/sqlite"
	"github.com/safari3308/adguard-portal/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	log.Printf("Starting AdGuard Portal server...")
	log.Printf("DoH Domain: %s", cfg.DOHDomain)
	log.Printf("Database path: %s", cfg.DBPath)

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to initialize sqlite: %v", err)
	}
	defer db.Close()
	log.Println("SQLite database initialized successfully.")

	// AGH Client
	aghClient := agh.NewClient(cfg.AGHURL, cfg.AGHUser, cfg.AGHPassword)

	// Repositories
	userRepo := sqlite.NewUserRepo(db)
	deviceRepo := sqlite.NewDeviceRepo(db)

	// Services
	deviceSvc := service.NewDeviceService(deviceRepo, aghClient, cfg.MaxDevicesPerUser, cfg.DOHDomain)
	if cfg.SigningCertPath != "" && cfg.SigningKeyPath != "" {
		signer, err := service.NewPKCS7SignerFromFiles(cfg.SigningCertPath, cfg.SigningKeyPath)
		if err != nil {
			log.Printf("WARN: Failed to initialize profile SSL signer: %v", err)
		} else {
			deviceSvc.SetSigner(signer)
			log.Println("Apple .mobileconfig SSL signer enabled.")
		}
	}

	// Auth Validator (Standard OIDC)
	authValidator := auth.NewValidator(cfg.OIDCIssuer, cfg.OIDCInternalURL, cfg.OIDCClientID, cfg.DevMode, userRepo)

	// Handlers
	userH := handler.NewUserHandler()
	deviceH := handler.NewDeviceHandler(deviceSvc)
	serviceH := handler.NewServiceHandler(aghClient, handler.ConfigResponse{
		DOHDomain:         cfg.DOHDomain,
		OIDCIssuer:        cfg.OIDCIssuer,
		OIDCClientID:      cfg.OIDCClientID,
		OIDCClientSecret:  cfg.OIDCClientSecret,
		OIDCScopes:        cfg.OIDCScopes,
		OIDCProviderName:  cfg.OIDCProviderName,
		MaxDevicesPerUser: cfg.MaxDevicesPerUser,
		DevMode:           cfg.DevMode,
	})

	router := appHTTP.NewRouter(appHTTP.RouterConfig{
		UserHandler:    userH,
		DeviceHandler:  deviceH,
		ServiceHandler: serviceH,
		AuthValidator:  authValidator,
		CORSOrigin:     cfg.CORSOrigin,
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server run loop
	go func() {
		log.Printf("Server listening on port %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced shutdown: %v", err)
	}

	log.Println("Server exited cleanly.")
}
