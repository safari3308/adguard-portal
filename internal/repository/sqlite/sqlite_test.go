package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/safari3308/adguard-portal/internal/domain"
)

func TestSQLiteRepository(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sub", "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// 1. User Repo Tests
	userRepo := NewUserRepo(db)

	// Get non-existent
	u, err := userRepo.GetByID(ctx, "non-existent")
	if err != nil {
		t.Fatalf("GetByID non-existent error: %v", err)
	}
	if u != nil {
		t.Errorf("expected nil for non-existent user, got %+v", u)
	}

	uByEmail, err := userRepo.GetByEmail(ctx, "unknown@example.com")
	if err != nil {
		t.Fatalf("GetByEmail non-existent error: %v", err)
	}
	if uByEmail != nil {
		t.Errorf("expected nil for non-existent email, got %+v", uByEmail)
	}

	// Upsert User
	testUser := &domain.User{
		ID:        "user-1",
		Email:     "user1@example.com",
		Name:      "User One",
		CreatedAt: time.Now().UTC(),
	}
	if err := userRepo.Upsert(ctx, testUser); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Read by ID
	fetchedUser, err := userRepo.GetByID(ctx, "user-1")
	if err != nil || fetchedUser == nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if fetchedUser.Name != "User One" {
		t.Errorf("expected User One, got %s", fetchedUser.Name)
	}

	// Read by Email
	fetchedByEmail, err := userRepo.GetByEmail(ctx, "user1@example.com")
	if err != nil || fetchedByEmail == nil {
		t.Fatalf("failed to get user by email: %v", err)
	}
	if fetchedByEmail.ID != "user-1" {
		t.Errorf("expected user-1, got %s", fetchedByEmail.ID)
	}

	// Update via Upsert (with zero CreatedAt to test fallback branch)
	testUser.Name = "User One Updated"
	testUser.CreatedAt = time.Time{}
	if err := userRepo.Upsert(ctx, testUser); err != nil {
		t.Fatalf("Update Upsert failed: %v", err)
	}
	fetchedUser, _ = userRepo.GetByID(ctx, "user-1")
	if fetchedUser.Name != "User One Updated" {
		t.Errorf("expected updated name, got %s", fetchedUser.Name)
	}

	// 2. Device Repo Tests
	deviceRepo := NewDeviceRepo(db)

	// Count initially
	count, err := deviceRepo.CountByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("CountByUserID failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 devices, got %d", count)
	}

	// Create Device
	d1 := &domain.Device{
		ID:                   "dev-1",
		UserID:               "user-1",
		Name:                 "My Laptop",
		ClientToken:          "tok-1234",
		FilteringEnabled:     true,
		ParentalEnabled:      false,
		SafeSearchEnabled:    true,
		BlockedServices:      []string{"youtube", "tiktok"},
		CustomBlockedDomains: []string{"bad.com"},
		CustomAllowedDomains: []string{"good.com"},
	}
	if err := deviceRepo.Create(ctx, d1); err != nil {
		t.Fatalf("deviceRepo.Create failed: %v", err)
	}

	// Count after create
	count, _ = deviceRepo.CountByUserID(ctx, "user-1")
	if count != 1 {
		t.Errorf("expected 1 device, got %d", count)
	}

	// GetByID
	gotD1, err := deviceRepo.GetByID(ctx, "dev-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if gotD1.Name != "My Laptop" || gotD1.ClientToken != "tok-1234" {
		t.Errorf("unexpected device data: %+v", gotD1)
	}
	if len(gotD1.BlockedServices) != 2 || gotD1.BlockedServices[0] != "youtube" {
		t.Errorf("unexpected blocked services: %+v", gotD1.BlockedServices)
	}

	// GetByToken
	gotByTok, err := deviceRepo.GetByToken(ctx, "tok-1234")
	if err != nil {
		t.Fatalf("GetByToken failed: %v", err)
	}
	if gotByTok.ID != "dev-1" {
		t.Errorf("expected dev-1, got %s", gotByTok.ID)
	}

	// GetByToken not found
	_, err = deviceRepo.GetByToken(ctx, "nonexistent-token")
	if err != domain.ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}

	// GetByID not found
	_, err = deviceRepo.GetByID(ctx, "nonexistent-id")
	if err != domain.ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}

	// Create second device
	d2 := &domain.Device{
		ID:          "dev-2",
		UserID:      "user-1",
		Name:        "My Phone",
		ClientToken: "tok-5678",
	}
	if err := deviceRepo.Create(ctx, d2); err != nil {
		t.Fatalf("failed to create d2: %v", err)
	}

	// ListByUserID
	devices, err := deviceRepo.ListByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUserID failed: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}

	// Update device
	gotD1.Name = "My Work Laptop"
	gotD1.ParentalEnabled = true
	gotD1.BlockedServices = []string{"netflix"}
	if err := deviceRepo.Update(ctx, gotD1); err != nil {
		t.Fatalf("Update device failed: %v", err)
	}

	updatedD1, _ := deviceRepo.GetByID(ctx, "dev-1")
	if updatedD1.Name != "My Work Laptop" || !updatedD1.ParentalEnabled {
		t.Errorf("update was not persisted: %+v", updatedD1)
	}

	// Update non-existent device -> should return ErrDeviceNotFound
	fakeDevice := &domain.Device{ID: "fake-id"}
	if err := deviceRepo.Update(ctx, fakeDevice); err != domain.ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound on update fake device, got %v", err)
	}

	// Delete device
	if err := deviceRepo.Delete(ctx, "dev-2"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	count, _ = deviceRepo.CountByUserID(ctx, "user-1")
	if count != 1 {
		t.Errorf("expected 1 device remaining, got %d", count)
	}

	// Delete non-existent device -> should return ErrDeviceNotFound
	if err := deviceRepo.Delete(ctx, "dev-2"); err != domain.ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound on delete nonexistent, got %v", err)
	}
}
