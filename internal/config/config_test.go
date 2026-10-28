package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadAndValidate(t *testing.T) {
	// Clean env before test
	envVars := []string{
		"PORT", "DB_PATH", "DOH_DOMAIN", "AGH_URL", "AGH_USER", "AGH_PASSWORD",
		"OIDC_ISSUER", "OIDC_CLIENT_ID", "OIDC_CLIENT_SECRET", "OIDC_SCOPES", "OIDC_PROVIDER_NAME",
		"ZITADEL_ISSUER", "ZITADEL_CLIENT_ID", "ZITADEL_CLIENT_SECRET", "MAX_DEVICES_PER_USER",
		"CORS_ORIGIN", "DEV_MODE",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}

	// 1. Defaults
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.DBPath != "data/adguard-portal.db" {
		t.Errorf("expected default DBPath, got %s", cfg.DBPath)
	}
	if cfg.DOHDomain != "localhost" {
		t.Errorf("expected doh domain localhost, got %s", cfg.DOHDomain)
	}
	if cfg.MaxDevicesPerUser != 10 {
		t.Errorf("expected 10 max devices, got %d", cfg.MaxDevicesPerUser)
	}
	if cfg.DevMode != false {
		t.Errorf("expected devMode false, got %v", cfg.DevMode)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("validate failed: %v", err)
	}

	// 2. Validate empty DOHDomain
	cfg.DOHDomain = ""
	if err := cfg.Validate(); err == nil {
		t.Errorf("expected error on empty DOHDomain, got nil")
	}

	// 3. Environment overrides (including legacy Zitadel fallback)
	os.Setenv("PORT", "9090")
	os.Setenv("ZITADEL_ISSUER", "https://zitadel.example.com")
	os.Setenv("ZITADEL_CLIENT_ID", "zitadel-client")
	os.Setenv("DEV_MODE", "true")
	os.Setenv("MAX_DEVICES_PER_USER", "25")
	os.Setenv("DOH_DOMAIN", "dns.example.com")

	cfg2, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg2.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg2.Port)
	}
	if cfg2.OIDCIssuer != "https://zitadel.example.com" {
		t.Errorf("expected zitadel issuer fallback, got %s", cfg2.OIDCIssuer)
	}
	if cfg2.OIDCClientID != "zitadel-client" {
		t.Errorf("expected zitadel client id fallback, got %s", cfg2.OIDCClientID)
	}
	if !cfg2.DevMode {
		t.Errorf("expected dev mode true")
	}
	if cfg2.MaxDevicesPerUser != 25 {
		t.Errorf("expected 25 max devices, got %d", cfg2.MaxDevicesPerUser)
	}

	// 4. Test OIDC_ISSUER taking precedence over ZITADEL_ISSUER
	os.Setenv("OIDC_ISSUER", "https://oidc.example.com")
	os.Setenv("OIDC_CLIENT_ID", "oidc-client-1")
	os.Setenv("OIDC_CLIENT_SECRET", "secret-123")
	cfg3, _ := Load()
	if cfg3.OIDCIssuer != "https://oidc.example.com" {
		t.Errorf("expected OIDC_ISSUER override, got %s", cfg3.OIDCIssuer)
	}
	if cfg3.OIDCClientID != "oidc-client-1" {
		t.Errorf("expected OIDC_CLIENT_ID override, got %s", cfg3.OIDCClientID)
	}
	if cfg3.OIDCClientSecret != "secret-123" {
		t.Errorf("expected OIDC_CLIENT_SECRET, got %s", cfg3.OIDCClientSecret)
	}

	// Clean up
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

func TestLoadDotEnv(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "test.env")
	content := `
# Comment line
PORT=3000
TEST_DOUBLE="double_quoted"
TEST_SINGLE='single_quoted'
INVALID_LINE
TEST_BOOL=true
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test env: %v", err)
	}

	os.Unsetenv("PORT")
	os.Unsetenv("TEST_DOUBLE")
	os.Unsetenv("TEST_SINGLE")
	os.Unsetenv("TEST_BOOL")

	if err := loadDotEnv(envFile); err != nil {
		t.Fatalf("loadDotEnv failed: %v", err)
	}

	if os.Getenv("PORT") != "3000" {
		t.Errorf("expected PORT=3000, got %s", os.Getenv("PORT"))
	}
	if os.Getenv("TEST_DOUBLE") != "double_quoted" {
		t.Errorf("expected TEST_DOUBLE=double_quoted, got %s", os.Getenv("TEST_DOUBLE"))
	}
	if os.Getenv("TEST_SINGLE") != "single_quoted" {
		t.Errorf("expected TEST_SINGLE=single_quoted, got %s", os.Getenv("TEST_SINGLE"))
	}

	// Non-existent file
	if err := loadDotEnv(filepath.Join(tmpDir, "nonexistent.env")); err == nil {
		t.Errorf("expected error for nonexistent file, got nil")
	}

	// Helpers
	os.Setenv("TEST_INT_INVALID", "abc")
	if val := getEnvAsInt("TEST_INT_INVALID", 42); val != 42 {
		t.Errorf("expected default 42 for invalid int, got %d", val)
	}
	os.Setenv("TEST_BOOL_INVALID", "notabool")
	if val := getEnvAsBool("TEST_BOOL_INVALID", true); val != true {
		t.Errorf("expected default true for invalid bool, got %v", val)
	}
}
