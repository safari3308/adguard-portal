package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port              int
	DBPath            string
	DOHDomain         string
	AGHURL            string
	AGHUser           string
	AGHPassword       string
	OIDCIssuer        string
	OIDCInternalURL   string
	OIDCClientID      string
	OIDCClientSecret  string
	OIDCScopes        string
	OIDCProviderName  string
	SigningCertPath   string
	SigningKeyPath    string
	MaxDevicesPerUser int
	CORSOrigin        string
	DevMode           bool
}

func Load() (*Config, error) {
	// Automatically load .env if present
	_ = loadDotEnv(".env")

	port := getEnvAsInt("PORT", 8080)
	dbPath := getEnv("DB_PATH", "data/adguard-portal.db")
	dohDomain := getEnv("DOH_DOMAIN", "localhost")
	aghURL := getEnv("AGH_URL", "http://127.0.0.1:3000")
	aghUser := getEnv("AGH_USER", "")
	aghPassword := getEnv("AGH_PASSWORD", "")

	// Generic standard OIDC configuration (with backwards compatible fallbacks)
	oidcIssuer := getEnv("OIDC_ISSUER", getEnv("ZITADEL_ISSUER", ""))
	oidcInternalURL := getEnv("OIDC_INTERNAL_URL", "")
	oidcClientID := getEnv("OIDC_CLIENT_ID", getEnv("ZITADEL_CLIENT_ID", ""))
	oidcClientSecret := getEnv("OIDC_CLIENT_SECRET", getEnv("ZITADEL_CLIENT_SECRET", ""))
	oidcScopes := getEnv("OIDC_SCOPES", "openid profile email")
	oidcProviderName := getEnv("OIDC_PROVIDER_NAME", "Single Sign-On")

	signingCertPath := getEnv("SIGNING_CERT_PATH", "")
	signingKeyPath := getEnv("SIGNING_KEY_PATH", "")

	maxDevices := getEnvAsInt("MAX_DEVICES_PER_USER", 10)
	corsOrigin := getEnv("CORS_ORIGIN", "*")
	devMode := getEnvAsBool("DEV_MODE", false)

	return &Config{
		Port:              port,
		DBPath:            dbPath,
		DOHDomain:         dohDomain,
		AGHURL:            aghURL,
		AGHUser:           aghUser,
		AGHPassword:       aghPassword,
		OIDCIssuer:        oidcIssuer,
		OIDCInternalURL:   oidcInternalURL,
		OIDCClientID:      oidcClientID,
		OIDCClientSecret:  oidcClientSecret,
		OIDCScopes:        oidcScopes,
		OIDCProviderName:  oidcProviderName,
		SigningCertPath:   signingCertPath,
		SigningKeyPath:    signingKeyPath,
		MaxDevicesPerUser: maxDevices,
		CORSOrigin:        corsOrigin,
		DevMode:           devMode,
	}, nil
}

func (c *Config) Validate() error {
	if c.DOHDomain == "" {
		return fmt.Errorf("DOH_DOMAIN cannot be empty")
	}
	return nil
}

// loadDotEnv parses a .env file and sets variables if they are not already set in the process environment.
func loadDotEnv(filepath string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return err // file does not exist, ignore
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Strip surrounding single or double quotes
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}

		// Only set if not already explicitly set in environment
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
	return scanner.Err()
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
