package agh

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/safari3308/adguard-portal/internal/domain"
)

func TestAGHClient(t *testing.T) {
	ctx := context.Background()

	var (
		updateCalled bool
		addCalled    bool
		deleteCalled bool
		accessList   = aghAccessList{
			AllowedClients: []string{"client-1"},
		}
	)

	// Create test server simulating AdGuard Home API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "admin" || p != "secret" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/control/clients/update":
			updateCalled = true
			var body aghUpdatePayload
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Name == "FailDevice (fail-token)" || body.Name == "BadDevice (bad-token)" {
				http.Error(w, "Not found", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)

		case "/control/clients/add":
			addCalled = true
			var body aghClientPayload
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Name == "BadDevice (bad-token)" {
				http.Error(w, "Conflict", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)

		case "/control/clients/delete":
			deleteCalled = true
			w.WriteHeader(http.StatusOK)

		case "/control/access/list":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(accessList)

		case "/control/access/set":
			var body aghAccessList
			_ = json.NewDecoder(r.Body).Decode(&body)
			accessList = body
			w.WriteHeader(http.StatusOK)

		case "/control/blocked_services/all":
			w.Header().Set("Content-Type", "application/json")
			// Return modern struct format
			resp := map[string]any{
				"blocked_services": []map[string]string{
					{"id": "discord", "name": "Discord", "icon": "discord.svg"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "admin", "secret")

	// 1. SyncDevice with successful update
	dev1 := &domain.Device{
		Name:                 "My Device",
		ClientToken:          "client-1",
		FilteringEnabled:     true,
		ParentalEnabled:      false,
		SafeSearchEnabled:    true,
		BlockedServices:      []string{"youtube"},
		CustomBlockedDomains: []string{"ads.example.com"},
		CustomAllowedDomains: []string{"cdn.example.com"},
	}
	if err := client.SyncDevice(ctx, dev1); err != nil {
		t.Fatalf("SyncDevice failed: %v", err)
	}
	if !updateCalled {
		t.Errorf("expected update to be called")
	}

	// 2. SyncDevice with fallback to add
	dev2 := &domain.Device{
		Name:                 "FailDevice",
		ClientToken:          "fail-token",
		FilteringEnabled:     false,
		CustomBlockedDomains: []string{"track.com"},
	}
	if err := client.SyncDevice(ctx, dev2); err != nil {
		t.Fatalf("SyncDevice add fallback failed: %v", err)
	}
	if !addCalled {
		t.Errorf("expected add to be called on update failure")
	}

	// 3. SyncDevice with total failure
	devBad := &domain.Device{
		Name:        "BadDevice",
		ClientToken: "bad-token",
	}
	if err := client.SyncDevice(ctx, devBad); err == nil {
		t.Errorf("expected SyncDevice to fail on bad client, got nil")
	}

	// 4. EnsureAllowedClient (client-1 is already in list)
	if err := client.EnsureAllowedClient(ctx, "client-1"); err != nil {
		t.Fatalf("EnsureAllowedClient failed: %v", err)
	}

	// 5. DeleteDevice
	if err := client.DeleteDevice(ctx, "fail-token", "FailDevice"); err != nil {
		t.Fatalf("DeleteDevice failed: %v", err)
	}
	if !deleteCalled {
		t.Errorf("expected delete to be called")
	}

	// 6. RemoveAllowedClient
	if err := client.RemoveAllowedClient(ctx, "client-1"); err != nil {
		t.Fatalf("RemoveAllowedClient failed: %v", err)
	}

	// 7. GetBlockedServices (structured response)
	services, err := client.GetBlockedServices(ctx)
	if err != nil {
		t.Fatalf("GetBlockedServices failed: %v", err)
	}
	if len(services) != 1 || services[0].ID != "discord" {
		t.Errorf("unexpected services response: %+v", services)
	}
}

func TestAGHClientFallbackAndHelpers(t *testing.T) {
	ctx := context.Background()

	// 1. Test fallback when server returns string array
	tsArray := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]string{"twitter", "reddit_app"})
	}))
	defer tsArray.Close()

	clientArray := NewClient(tsArray.URL, "", "")
	services, err := clientArray.GetBlockedServices(ctx)
	if err != nil {
		t.Fatalf("GetBlockedServices array failed: %v", err)
	}
	if len(services) != 2 || services[1].Name != "Reddit App" {
		t.Errorf("unexpected formatted name: %+v", services)
	}

	// 2. Test fallback to defaultBlockedServices on error / non-200
	tsErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
	}))
	defer tsErr.Close()

	clientErr := NewClient(tsErr.URL, "", "")
	servicesDef, err := clientErr.GetBlockedServices(ctx)
	if err != nil {
		t.Fatalf("GetBlockedServices fallback failed: %v", err)
	}
	if len(servicesDef) == 0 {
		t.Errorf("expected default blocked services on error")
	}

	// 3. Test compileRules
	rules := compileRules([]string{"bad.com", " evil.org "}, []string{"good.com"})
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}
	if rules[0] != "||bad.com^" || rules[1] != "||evil.org^" || rules[2] != "@@||good.com^" {
		t.Errorf("unexpected rule formatting: %v", rules)
	}
}
