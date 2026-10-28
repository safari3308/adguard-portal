package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceLimitReached  = errors.New("device limit reached")
	ErrInvalidDomainRules  = errors.New("invalid domain rule format")
	ErrMaxRulesExceeded    = errors.New("maximum number of custom rules (50) exceeded")
	ErrUnauthorizedDevice  = errors.New("unauthorized device access")
)

type Device struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"user_id"`
	Name                 string    `json:"name"`
	ClientToken          string    `json:"client_token"`
	FilteringEnabled     bool      `json:"filtering_enabled"`
	ParentalEnabled      bool      `json:"parental_enabled"`
	SafeSearchEnabled    bool      `json:"safe_search_enabled"`
	BlockedServices      []string  `json:"blocked_services"`
	CustomBlockedDomains []string  `json:"custom_blocked_domains"`
	CustomAllowedDomains []string  `json:"custom_allowed_domains"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type CreateDeviceInput struct {
	Name string `json:"name"`
}

type UpdateDeviceInput struct {
	Name                 *string   `json:"name,omitempty"`
	FilteringEnabled     *bool     `json:"filtering_enabled,omitempty"`
	ParentalEnabled      *bool     `json:"parental_enabled,omitempty"`
	SafeSearchEnabled    *bool     `json:"safe_search_enabled,omitempty"`
	BlockedServices      *[]string `json:"blocked_services,omitempty"`
	CustomBlockedDomains *[]string `json:"custom_blocked_domains,omitempty"`
	CustomAllowedDomains *[]string `json:"custom_allowed_domains,omitempty"`
}

type DeviceRepository interface {
	Create(ctx context.Context, device *Device) error
	GetByID(ctx context.Context, id string) (*Device, error)
	GetByToken(ctx context.Context, token string) (*Device, error)
	ListByUserID(ctx context.Context, userID string) ([]*Device, error)
	Update(ctx context.Context, device *Device) error
	Delete(ctx context.Context, id string) error
	CountByUserID(ctx context.Context, userID string) (int, error)
}
