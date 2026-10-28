package service

import (
	"context"
	"strings"
	"testing"

	"github.com/safari3308/adguard-portal/internal/domain"
	"github.com/safari3308/adguard-portal/internal/repository/sqlite"
)

type mockAGHClient struct {
	syncErr   error
	deleteErr error
}

func (m *mockAGHClient) SyncDevice(ctx context.Context, device *domain.Device) error {
	return m.syncErr
}

func (m *mockAGHClient) DeleteDevice(ctx context.Context, clientToken, deviceName string) error {
	return m.deleteErr
}

func (m *mockAGHClient) EnsureAllowedClient(ctx context.Context, clientToken string) error {
	return nil
}

func (m *mockAGHClient) RemoveAllowedClient(ctx context.Context, clientToken string) error {
	return nil
}

func (m *mockAGHClient) GetBlockedServices(ctx context.Context) ([]domain.BlockedService, error) {
	return nil, nil
}

func TestDeviceLifecycle(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open memory sqlite: %v", err)
	}
	defer db.Close()

	userRepo := sqlite.NewUserRepo(db)
	deviceRepo := sqlite.NewDeviceRepo(db)

	testUser := &domain.User{
		ID:    "user_123",
		Email: "test@example.com",
		Name:  "Test User",
	}
	_ = userRepo.Upsert(ctx, testUser)

	aghClient := &mockAGHClient{}
	svc := NewDeviceService(deviceRepo, aghClient, 2, "dns.test.local")

	// 1. Create Device
	d1, err := svc.CreateDevice(ctx, testUser.ID, domain.CreateDeviceInput{Name: "iPhone 15"})
	if err != nil {
		t.Fatalf("failed to create device: %v", err)
	}
	if d1.Name != "iPhone 15" {
		t.Errorf("expected name iPhone 15, got %s", d1.Name)
	}

	// 2. GetDevice
	gotD1, err := svc.GetDevice(ctx, testUser.ID, d1.ID)
	if err != nil || gotD1 == nil {
		t.Fatalf("failed to get device: %v", err)
	}

	// GetDevice unauthorized
	_, err = svc.GetDevice(ctx, "other_user", d1.ID)
	if err != domain.ErrUnauthorizedDevice {
		t.Errorf("expected ErrUnauthorizedDevice, got %v", err)
	}

	// GetDevice nonexistent
	_, err = svc.GetDevice(ctx, testUser.ID, "nonexistent-id")
	if err != domain.ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}

	// 3. ListDevices
	list, err := svc.ListDevices(ctx, testUser.ID)
	if err != nil || len(list) != 1 {
		t.Errorf("expected 1 device in list, got %v", list)
	}

	// 4. Update Device
	newName := "iPhone 15 Pro"
	parentalTrue := true
	safeSearchTrue := true
	filteringFalse := false
	services := []string{"tiktok", "youtube"}
	blockedDomains := []string{"https://tracker.bad.com/"}
	allowedDomains := []string{"http://cdn.good.com"}

	d1Updated, err := svc.UpdateDevice(ctx, testUser.ID, d1.ID, domain.UpdateDeviceInput{
		Name:                 &newName,
		FilteringEnabled:     &filteringFalse,
		ParentalEnabled:      &parentalTrue,
		SafeSearchEnabled:    &safeSearchTrue,
		BlockedServices:      &services,
		CustomBlockedDomains: &blockedDomains,
		CustomAllowedDomains: &allowedDomains,
	})
	if err != nil {
		t.Fatalf("failed to update device: %v", err)
	}
	if d1Updated.Name != "iPhone 15 Pro" || !d1Updated.ParentalEnabled || !d1Updated.SafeSearchEnabled {
		t.Errorf("update fields failed: %+v", d1Updated)
	}
	if len(d1Updated.CustomBlockedDomains) != 1 || d1Updated.CustomBlockedDomains[0] != "tracker.bad.com" {
		t.Errorf("expected cleaned domain tracker.bad.com, got %v", d1Updated.CustomBlockedDomains)
	}

	// Update unauthorized
	_, err = svc.UpdateDevice(ctx, "other_user", d1.ID, domain.UpdateDeviceInput{})
	if err != domain.ErrUnauthorizedDevice {
		t.Errorf("expected ErrUnauthorizedDevice, got %v", err)
	}

	// Update nonexistent
	_, err = svc.UpdateDevice(ctx, testUser.ID, "nonexistent", domain.UpdateDeviceInput{})
	if err != domain.ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}

	// 5. Max devices limit check
	d2, err := svc.CreateDevice(ctx, testUser.ID, domain.CreateDeviceInput{Name: "MacBook"})
	if err != nil {
		t.Fatalf("failed to create second device: %v", err)
	}

	_, err = svc.CreateDevice(ctx, testUser.ID, domain.CreateDeviceInput{Name: "iPad"})
	if err != domain.ErrDeviceLimitReached {
		t.Errorf("expected ErrDeviceLimitReached, got %v", err)
	}

	// 6. Generate MobileConfig
	configXML, err := svc.GenerateMobileConfig(ctx, testUser.ID, d1.ID)
	if err != nil || len(configXML) == 0 {
		t.Fatalf("failed to generate mobileconfig: %v", err)
	}

	// Test signed MobileConfig with signer
	certPEM, keyPEM := generateTestCertAndKey(t, true)
	signer, err := NewPKCS7Signer(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}
	svc.SetSigner(signer)
	signedConfig, err := svc.GenerateMobileConfig(ctx, testUser.ID, d1.ID)
	if err != nil || len(signedConfig) <= len(configXML) {
		t.Fatalf("failed to generate signed mobileconfig: %v", err)
	}
	svc.SetSigner(nil)

	// Generate MobileConfig unauthorized
	_, err = svc.GenerateMobileConfig(ctx, "other_user", d1.ID)
	if err != domain.ErrUnauthorizedDevice {
		t.Errorf("expected ErrUnauthorizedDevice, got %v", err)
	}

	// 7. Delete device
	// Delete unauthorized
	err = svc.DeleteDevice(ctx, "other_user", d1.ID)
	if err != domain.ErrUnauthorizedDevice {
		t.Errorf("expected ErrUnauthorizedDevice, got %v", err)
	}

	// Delete nonexistent
	err = svc.DeleteDevice(ctx, testUser.ID, "nonexistent")
	if err != domain.ErrDeviceNotFound {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}

	// Delete success
	if err := svc.DeleteDevice(ctx, testUser.ID, d1.ID); err != nil {
		t.Fatalf("failed to delete d1: %v", err)
	}
	if err := svc.DeleteDevice(ctx, testUser.ID, d2.ID); err != nil {
		t.Fatalf("failed to delete d2: %v", err)
	}

	// 8. Test domain sanitization edge cases
	// Invalid domain character
	badDomains := []string{"invalid domain!!"}
	_, err = sanitizeDomains(badDomains)
	if err == nil {
		t.Errorf("expected error for invalid domain format, got nil")
	}

	// Exceed max rules
	tooMany := make([]string, 51)
	for i := range tooMany {
		tooMany[i] = "example.com"
	}
	_, err = sanitizeDomains(tooMany)
	if err != domain.ErrMaxRulesExceeded {
		t.Errorf("expected ErrMaxRulesExceeded, got %v", err)
	}
}

func TestMobileConfigFormatting(t *testing.T) {
	dev := &domain.Device{
		ID:          "dev-test",
		Name:        "Test Mobile",
		ClientToken: "u-abcdef",
	}
	data, err := GenerateMobileConfig(dev, "dns.example.com")
	if err != nil {
		t.Fatalf("GenerateMobileConfig failed: %v", err)
	}
	xmlStr := string(data)
	if !strings.Contains(xmlStr, "https://dns.example.com/dns-query/u-abcdef") {
		t.Errorf("expected DoH URL in mobileconfig, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, "dns.example.com") {
		t.Errorf("expected ServerAddresses in mobileconfig, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, "<key>OnDemandRules</key>") {
		t.Errorf("expected OnDemandRules in mobileconfig, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, "<key>URLStringProbe</key>") {
		t.Errorf("expected URLStringProbe in mobileconfig, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, "<key>ProhibitDisablement</key>") {
		t.Errorf("expected ProhibitDisablement in mobileconfig, got: %s", xmlStr)
	}
}
