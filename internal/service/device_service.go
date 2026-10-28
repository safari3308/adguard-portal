package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/safari3308/adguard-portal/internal/domain"
)

var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

type DeviceService struct {
	repo       domain.DeviceRepository
	agh        domain.AGHClient
	maxDevices int
	dohDomain  string
	signer     ProfileSigner
}

func NewDeviceService(repo domain.DeviceRepository, agh domain.AGHClient, maxDevices int, dohDomain string) *DeviceService {
	return &DeviceService{
		repo:       repo,
		agh:        agh,
		maxDevices: maxDevices,
		dohDomain:  dohDomain,
	}
}

func (s *DeviceService) SetSigner(signer ProfileSigner) {
	s.signer = signer
}

func (s *DeviceService) CreateDevice(ctx context.Context, userID string, input domain.CreateDeviceInput) (*domain.Device, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = "My Device"
	}

	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check device count: %w", err)
	}
	if count >= s.maxDevices {
		return nil, domain.ErrDeviceLimitReached
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	device := &domain.Device{
		ID:                   uuid.NewString(),
		UserID:               userID,
		Name:                 name,
		ClientToken:          token,
		FilteringEnabled:     true,
		ParentalEnabled:      false,
		SafeSearchEnabled:    false,
		BlockedServices:      []string{},
		CustomBlockedDomains: []string{},
		CustomAllowedDomains: []string{},
	}

	if err := s.repo.Create(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to save device: %w", err)
	}

	// Sync to AdGuard Home (non-fatal if AGH is momentarily unreachable, but logged)
	if s.agh != nil {
		_ = s.agh.SyncDevice(ctx, device)
	}

	return device, nil
}

func (s *DeviceService) GetDevice(ctx context.Context, userID, deviceID string) (*domain.Device, error) {
	device, err := s.repo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if device.UserID != userID {
		return nil, domain.ErrUnauthorizedDevice
	}
	return device, nil
}

func (s *DeviceService) ListDevices(ctx context.Context, userID string) ([]*domain.Device, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *DeviceService) UpdateDevice(ctx context.Context, userID, deviceID string, input domain.UpdateDeviceInput) (*domain.Device, error) {
	device, err := s.repo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if device.UserID != userID {
		return nil, domain.ErrUnauthorizedDevice
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name != "" {
			device.Name = name
		}
	}
	if input.FilteringEnabled != nil {
		device.FilteringEnabled = *input.FilteringEnabled
	}
	if input.ParentalEnabled != nil {
		device.ParentalEnabled = *input.ParentalEnabled
	}
	if input.SafeSearchEnabled != nil {
		device.SafeSearchEnabled = *input.SafeSearchEnabled
	}
	if input.BlockedServices != nil {
		device.BlockedServices = *input.BlockedServices
	}
	if input.CustomBlockedDomains != nil {
		cleaned, err := sanitizeDomains(*input.CustomBlockedDomains)
		if err != nil {
			return nil, err
		}
		device.CustomBlockedDomains = cleaned
	}
	if input.CustomAllowedDomains != nil {
		cleaned, err := sanitizeDomains(*input.CustomAllowedDomains)
		if err != nil {
			return nil, err
		}
		device.CustomAllowedDomains = cleaned
	}

	if err := s.repo.Update(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to update device in db: %w", err)
	}

	if s.agh != nil {
		_ = s.agh.SyncDevice(ctx, device)
	}

	return device, nil
}

func (s *DeviceService) DeleteDevice(ctx context.Context, userID, deviceID string) error {
	device, err := s.repo.GetByID(ctx, deviceID)
	if err != nil {
		return err
	}
	if device.UserID != userID {
		return domain.ErrUnauthorizedDevice
	}

	if err := s.repo.Delete(ctx, deviceID); err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	if s.agh != nil {
		_ = s.agh.DeleteDevice(ctx, device.ClientToken, device.Name)
	}

	return nil
}

func (s *DeviceService) GenerateMobileConfig(ctx context.Context, userID, deviceID string) ([]byte, error) {
	device, err := s.GetDevice(ctx, userID, deviceID)
	if err != nil {
		return nil, err
	}
	raw, err := GenerateMobileConfig(device, s.dohDomain)
	if err != nil {
		return nil, err
	}
	if s.signer != nil {
		signed, err := s.signer.Sign(raw)
		if err != nil {
			fmt.Printf("ERROR: Failed to sign mobileconfig for device %s: %v\n", deviceID, err)
		} else {
			return signed, nil
		}
	}
	return raw, nil
}

func generateToken() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("u-%s", hex.EncodeToString(b)), nil
}

func sanitizeDomains(domains []string) ([]string, error) {
	if len(domains) > 50 {
		return nil, domain.ErrMaxRulesExceeded
	}

	out := make([]string, 0, len(domains))
	seen := make(map[string]bool)

	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		// Strip common prefixes like http:// or https:// if pasted by mistake
		d = strings.TrimPrefix(d, "https://")
		d = strings.TrimPrefix(d, "http://")
		d = strings.TrimRight(d, "/")

		if !domainRegex.MatchString(d) {
			return nil, fmt.Errorf("%w: %s", domain.ErrInvalidDomainRules, d)
		}

		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}

	return out, nil
}
