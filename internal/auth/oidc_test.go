package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/safari3308/adguard-portal/internal/domain"
)

type mockUserRepo struct {
	users map[string]*domain.User
}

func (m *mockUserRepo) Upsert(ctx context.Context, u *domain.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return m.users[id], nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func TestOIDCValidator(t *testing.T) {
	// 1. Generate RSA key for test
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	nStr := base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes())
	eBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(eBytes, uint32(publicKey.E))
	// strip leading zero bytes
	for len(eBytes) > 0 && eBytes[0] == 0 {
		eBytes = eBytes[1:]
	}
	eStr := base64.RawURLEncoding.EncodeToString(eBytes)

	// 2. Setup mock OIDC server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"jwks_uri":          fmt.Sprintf("http://%s/keys", r.Host),
				"userinfo_endpoint": fmt.Sprintf("http://%s/userinfo", r.Host),
			})
		case "/userinfo":
			authH := r.Header.Get("Authorization")
			if authH == "Bearer opaque-token-123" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]string{
					"sub":   "userinfo-sub-999",
					"email": "userinfo@example.com",
					"name":  "Userinfo User",
				})
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		case "/keys":
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"keys": []map[string]string{
					{
						"kid": "test-kid",
						"kty": "RSA",
						"alg": "RS256",
						"use": "sig",
						"n":   nStr,
						"e":   eStr,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	repo := &mockUserRepo{users: make(map[string]*domain.User)}
	validator := NewValidator(ts.URL, "test-client-id", true, repo)

	handler := validator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := UserFromContext(r.Context())
		if u == nil {
			http.Error(w, "missing user in context", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(u.Email))
	}))

	// Test 1: Missing auth header
	req1 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on missing auth header, got %d", rec1.Code)
	}

	// Test 2: Invalid header format
	req2 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req2.Header.Set("Authorization", "Basic 12345")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on invalid format, got %d", rec2.Code)
	}

	// Test 3: Dev Mode token
	req3 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req3.Header.Set("Authorization", "Bearer dev-user-admin")
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Errorf("expected 200 on dev mode token, got %d", rec3.Code)
	}
	if rec3.Body.String() != "dev-user-admin@example.com" {
		t.Errorf("unexpected dev user email: %s", rec3.Body.String())
	}

	// Test 4: Valid RS256 token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, &StandardOIDCClaims{
		Email: "alice@example.com",
		Name:  "Alice Example",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.URL,
			Subject:   "alice-sub-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	token.Header["kid"] = "test-kid"
	tokenStr, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req4 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req4.Header.Set("Authorization", "Bearer "+tokenStr)
	rec4 := httptest.NewRecorder()
	handler.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusOK {
		t.Errorf("expected 200 on valid token, got %d: %s", rec4.Code, rec4.Body.String())
	}
	if rec4.Body.String() != "alice@example.com" {
		t.Errorf("expected alice@example.com, got %s", rec4.Body.String())
	}

	// Test 5: Fallback username logic when email and name are empty
	tokenNoInfo := jwt.NewWithClaims(jwt.SigningMethodRS256, &StandardOIDCClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.URL,
			Subject:   "bob-sub-456",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	tokenNoInfo.Header["kid"] = "test-kid"
	tokenNoInfoStr, _ := tokenNoInfo.SignedString(privateKey)

	req5 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req5.Header.Set("Authorization", "Bearer "+tokenNoInfoStr)
	rec5 := httptest.NewRecorder()
	handler.ServeHTTP(rec5, req5)
	if rec5.Code != http.StatusOK {
		t.Errorf("expected 200 on fallback user token, got %d", rec5.Code)
	}

	// Test 6: Expired token
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodRS256, &StandardOIDCClaims{
		Email: "charlie@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.URL,
			Subject:   "charlie-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	})
	expiredToken.Header["kid"] = "test-kid"
	expiredTokenStr, _ := expiredToken.SignedString(privateKey)

	req6 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req6.Header.Set("Authorization", "Bearer "+expiredTokenStr)
	rec6 := httptest.NewRecorder()
	handler.ServeHTTP(rec6, req6)
	if rec6.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on expired token, got %d", rec6.Code)
	}

	// Test 7: Unknown kid
	unknownKidToken := jwt.NewWithClaims(jwt.SigningMethodRS256, &StandardOIDCClaims{
		Email: "unknown@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.URL,
			Subject:   "unknown-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	unknownKidToken.Header["kid"] = "unknown-kid"
	unknownKidTokenStr, _ := unknownKidToken.SignedString(privateKey)

	req7 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req7.Header.Set("Authorization", "Bearer "+unknownKidTokenStr)
	rec7 := httptest.NewRecorder()
	handler.ServeHTTP(rec7, req7)
	if rec7.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on unknown kid token, got %d", rec7.Code)
	}

	// Test 8: Missing kid
	noKidToken := jwt.NewWithClaims(jwt.SigningMethodRS256, &StandardOIDCClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.URL,
			Subject:   "nokid",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	noKidTokenStr, _ := noKidToken.SignedString(privateKey)
	req8 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req8.Header.Set("Authorization", "Bearer "+noKidTokenStr)
	rec8 := httptest.NewRecorder()
	handler.ServeHTTP(rec8, req8)
	if rec8.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on missing kid, got %d", rec8.Code)
	}

	// Test 9: Empty issuer discoverJWKSURI
	vNoIssuer := NewValidator("", "client", false, repo)
	if _, err := vNoIssuer.discoverJWKSURI(); err == nil {
		t.Errorf("expected error for empty issuer in discoverJWKSURI, got nil")
	}

	// Test 10: Fallback when .well-known/openid-configuration is 404
	tsFallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer tsFallback.Close()

	vFallback := NewValidator(tsFallback.URL, "client", false, repo)
	uri, err := vFallback.discoverJWKSURI()
	if err != nil || uri != tsFallback.URL+"/.well-known/jwks.json" {
		t.Errorf("expected fallback jwks uri, got %s, err: %v", uri, err)
	}

	// Test 11: Opaque token via Userinfo endpoint (success + cache)
	reqOpaque := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	reqOpaque.Header.Set("Authorization", "Bearer opaque-token-123")
	recOpaque := httptest.NewRecorder()
	handler.ServeHTTP(recOpaque, reqOpaque)
	if recOpaque.Code != http.StatusOK {
		t.Errorf("expected 200 on opaque token via userinfo, got %d: %s", recOpaque.Code, recOpaque.Body.String())
	}
	if recOpaque.Body.String() != "userinfo@example.com" {
		t.Errorf("expected userinfo@example.com, got %s", recOpaque.Body.String())
	}

	// Test cached opaque token
	recOpaqueCached := httptest.NewRecorder()
	handler.ServeHTTP(recOpaqueCached, reqOpaque)
	if recOpaqueCached.Code != http.StatusOK {
		t.Errorf("expected 200 on cached opaque token, got %d", recOpaqueCached.Code)
	}

	// Test 12: Bad opaque token -> 401
	reqBadOpaque := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	reqBadOpaque.Header.Set("Authorization", "Bearer bad-opaque-token")
	recBadOpaque := httptest.NewRecorder()
	handler.ServeHTTP(recBadOpaque, reqBadOpaque)
	if recBadOpaque.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on bad opaque token, got %d", recBadOpaque.Code)
	}
}
