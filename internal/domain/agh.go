package domain

import "context"

type BlockedService struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Rules []string `json:"rules,omitempty"`
	Icon  string   `json:"icon,omitempty"`
}

type AGHClient interface {
	SyncDevice(ctx context.Context, device *Device) error
	DeleteDevice(ctx context.Context, clientToken string, deviceName string) error
	GetBlockedServices(ctx context.Context) ([]BlockedService, error)
	EnsureAllowedClient(ctx context.Context, clientToken string) error
	RemoveAllowedClient(ctx context.Context, clientToken string) error
}
