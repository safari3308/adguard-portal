package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/safari3308/adguard-portal/internal/domain"
)

// StandardOIDCClaims handles compliant OIDC ID and Access tokens across providers.
type StandardOIDCClaims struct {
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	GivenName         string `json:"given_name"`
	jwt.RegisteredClaims
}

type openIDConfiguration struct {
	Issuer           string `json:"issuer"`
	JwksURI          string `json:"jwks_uri"`
	UserinfoEndpoint string `json:"userinfo_endpoint"`
}

type jwksKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

type cachedUserinfo struct {
	claims    *StandardOIDCClaims
	expiresAt time.Time
}

type Validator struct {
	issuer           string
	internalURL      string
	clientID         string
	devMode          bool
	userRepo         domain.UserRepository
	jwksURI          string
	userinfoEndpoint string
	jwksMu           sync.RWMutex
	keys             map[string]*rsa.PublicKey
	lastFetch        time.Time
	userinfoCache    map[string]cachedUserinfo
	userinfoMu       sync.RWMutex
}

func NewValidator(issuer, internalURL, clientID string, devMode bool, userRepo domain.UserRepository) *Validator {
	return &Validator{
		issuer:        strings.TrimRight(issuer, "/"),
		internalURL:   strings.TrimRight(internalURL, "/"),
		clientID:      clientID,
		devMode:       devMode,
		userRepo:      userRepo,
		keys:          make(map[string]*rsa.PublicKey),
		userinfoCache: make(map[string]cachedUserinfo),
	}
}

func (v *Validator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// Support Mock/Dev Mode token for rapid local testing
		if v.devMode && strings.HasPrefix(tokenString, "dev-user-") {
			user := &domain.User{
				ID:    tokenString,
				Email: fmt.Sprintf("%s@example.com", tokenString),
				Name:  "Developer User",
			}
			_ = v.userRepo.Upsert(r.Context(), user)
			ctx := WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		claims, err := v.verifyToken(tokenString)
		if err != nil {
			var userinfoErr error
			claims, userinfoErr = v.verifyViaUserinfo(r.Context(), tokenString)
			if userinfoErr != nil {
				http.Error(w, fmt.Sprintf(`{"error":"unauthorized: %s"}`, err.Error()), http.StatusUnauthorized)
				return
			}
		}

		userEmail := claims.Email
		if userEmail == "" {
			userEmail = claims.PreferredUsername
		}
		if userEmail == "" {
			userEmail = fmt.Sprintf("%s@unknown", claims.Subject)
		}

		userName := claims.Name
		if userName == "" {
			userName = claims.GivenName
		}
		if userName == "" {
			userName = strings.Split(userEmail, "@")[0]
		}

		user := &domain.User{
			ID:    claims.Subject,
			Email: userEmail,
			Name:  userName,
		}

		if err := v.userRepo.Upsert(r.Context(), user); err != nil {
			http.Error(w, `{"error":"failed to sync user profile"}`, http.StatusInternalServerError)
			return
		}

		ctx := WithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (v *Validator) verifyToken(tokenStr string) (*StandardOIDCClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &StandardOIDCClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, errors.New("token header missing 'kid'")
		}

		return v.getKey(kid)
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*StandardOIDCClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	if v.issuer != "" && !strings.HasPrefix(claims.Issuer, v.issuer) {
		return nil, fmt.Errorf("issuer mismatch: expected prefix %s, got %s", v.issuer, claims.Issuer)
	}

	return claims, nil
}

func (v *Validator) getKey(kid string) (*rsa.PublicKey, error) {
	v.jwksMu.RLock()
	key, exists := v.keys[kid]
	lastFetch := v.lastFetch
	v.jwksMu.RUnlock()

	if exists && time.Since(lastFetch) < 1*time.Hour {
		return key, nil
	}

	// Refresh JWKS from OIDC provider
	if err := v.refreshJWKS(); err != nil {
		if exists {
			return key, nil // fallback to cached key
		}
		return nil, fmt.Errorf("failed to fetch jwks: %w", err)
	}

	v.jwksMu.RLock()
	defer v.jwksMu.RUnlock()
	key, exists = v.keys[kid]
	if !exists {
		return nil, fmt.Errorf("key with kid %s not found in jwks", kid)
	}
	return key, nil
}

func (v *Validator) rewriteURL(originalURL string) string {
	if v.internalURL == "" || v.issuer == "" {
		return originalURL
	}
	if strings.HasPrefix(originalURL, v.issuer) {
		return v.internalURL + strings.TrimPrefix(originalURL, v.issuer)
	}
	return originalURL
}

func (v *Validator) discoverJWKSURI() (string, error) {
	if v.jwksURI != "" {
		return v.jwksURI, nil
	}
	baseURL := v.internalURL
	if baseURL == "" {
		baseURL = v.issuer
	}
	if baseURL == "" {
		return "", errors.New("OIDC_ISSUER not configured")
	}

	client := &http.Client{Timeout: 5 * time.Second}

	// 1. Standard OIDC Discovery: /.well-known/openid-configuration
	discoveryURL := fmt.Sprintf("%s/.well-known/openid-configuration", baseURL)
	resp, err := client.Get(discoveryURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var doc openIDConfiguration
		if err := json.NewDecoder(resp.Body).Decode(&doc); err == nil && doc.JwksURI != "" {
			v.jwksURI = v.rewriteURL(doc.JwksURI)
			if doc.UserinfoEndpoint != "" {
				v.userinfoEndpoint = v.rewriteURL(doc.UserinfoEndpoint)
			}
			return v.jwksURI, nil
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	// 2. Fallbacks for providers without discovery
	fallbacks := []string{
		fmt.Sprintf("%s/oauth/v2/keys", baseURL),
		fmt.Sprintf("%s/.well-known/jwks.json", baseURL),
		fmt.Sprintf("%s/keys", baseURL),
	}

	for _, u := range fallbacks {
		r, err := client.Head(u)
		if err == nil && (r.StatusCode == http.StatusOK || r.StatusCode == http.StatusMethodNotAllowed) {
			r.Body.Close()
			v.jwksURI = u
			return v.jwksURI, nil
		}
		if r != nil {
			r.Body.Close()
		}
	}

	// Default fallback
	v.jwksURI = fmt.Sprintf("%s/.well-known/jwks.json", baseURL)
	return v.jwksURI, nil
}

func (v *Validator) refreshJWKS() error {
	v.jwksMu.Lock()
	defer v.jwksMu.Unlock()

	jwksURL, err := v.discoverJWKSURI()
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(jwksURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	newKeys := make(map[string]*rsa.PublicKey)
	for _, k := range jwks.Keys {
		if k.Kty == "RSA" && k.N != "" && k.E != "" {
			pubKey, err := parseRSAPublicKey(k.N, k.E)
			if err == nil {
				newKeys[k.Kid] = pubKey
			}
		}
	}

	v.keys = newKeys
	v.lastFetch = time.Now()
	return nil
}

func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}

	var eInt uint64
	for _, b := range eBytes {
		eInt = (eInt << 8) | uint64(b)
	}

	n := new(big.Int).SetBytes(nBytes)
	return &rsa.PublicKey{
		N: n,
		E: int(eInt),
	}, nil
}

func (v *Validator) verifyViaUserinfo(ctx context.Context, tokenStr string) (*StandardOIDCClaims, error) {
	if tokenStr == "" {
		return nil, errors.New("empty token")
	}

	v.userinfoMu.RLock()
	if cached, ok := v.userinfoCache[tokenStr]; ok && time.Now().Before(cached.expiresAt) {
		v.userinfoMu.RUnlock()
		return cached.claims, nil
	}
	v.userinfoMu.RUnlock()

	if v.userinfoEndpoint == "" {
		_, _ = v.discoverJWKSURI()
	}
	if v.userinfoEndpoint == "" {
		baseURL := v.internalURL
		if baseURL == "" {
			baseURL = v.issuer
		}
		if baseURL == "" {
			return nil, errors.New("OIDC_ISSUER not configured")
		}
		v.userinfoEndpoint = fmt.Sprintf("%s/oidc/v1/userinfo", baseURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.userinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned status %d", resp.StatusCode)
	}

	var claims StandardOIDCClaims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}
	if claims.Subject == "" {
		return nil, errors.New("userinfo response missing subject")
	}

	v.userinfoMu.Lock()
	v.userinfoCache[tokenStr] = cachedUserinfo{
		claims:    &claims,
		expiresAt: time.Now().Add(2 * time.Minute),
	}
	v.userinfoMu.Unlock()

	return &claims, nil
}
