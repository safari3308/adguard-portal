package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/safari3308/adguard-portal/internal/domain"
)

type deviceRepo struct {
	db *DB
}

func NewDeviceRepo(db *DB) domain.DeviceRepository {
	return &deviceRepo{db: db}
}

func (r *deviceRepo) Create(ctx context.Context, d *domain.Device) error {
	query := `
	INSERT INTO devices (
		id, user_id, name, client_token,
		filtering_enabled, parental_enabled, safe_search_enabled,
		blocked_services, custom_blocked_domains, custom_allowed_domains,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	servicesJSON, _ := json.Marshal(d.BlockedServices)
	blockedJSON, _ := json.Marshal(d.CustomBlockedDomains)
	allowedJSON, _ := json.Marshal(d.CustomAllowedDomains)

	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		d.ID, d.UserID, d.Name, d.ClientToken,
		d.FilteringEnabled, d.ParentalEnabled, d.SafeSearchEnabled,
		string(servicesJSON), string(blockedJSON), string(allowedJSON),
		d.CreatedAt, d.UpdatedAt,
	)
	return err
}

func (r *deviceRepo) GetByID(ctx context.Context, id string) (*domain.Device, error) {
	query := `
	SELECT id, user_id, name, client_token,
	       filtering_enabled, parental_enabled, safe_search_enabled,
	       blocked_services, custom_blocked_domains, custom_allowed_domains,
	       created_at, updated_at
	FROM devices WHERE id = ?;
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanDevice(row)
}

func (r *deviceRepo) GetByToken(ctx context.Context, token string) (*domain.Device, error) {
	query := `
	SELECT id, user_id, name, client_token,
	       filtering_enabled, parental_enabled, safe_search_enabled,
	       blocked_services, custom_blocked_domains, custom_allowed_domains,
	       created_at, updated_at
	FROM devices WHERE client_token = ?;
	`
	row := r.db.QueryRowContext(ctx, query, token)
	return scanDevice(row)
}

func (r *deviceRepo) ListByUserID(ctx context.Context, userID string) ([]*domain.Device, error) {
	query := `
	SELECT id, user_id, name, client_token,
	       filtering_enabled, parental_enabled, safe_search_enabled,
	       blocked_services, custom_blocked_domains, custom_allowed_domains,
	       created_at, updated_at
	FROM devices WHERE user_id = ? ORDER BY created_at DESC;
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*domain.Device
	for rows.Next() {
		d, err := scanDeviceRow(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (r *deviceRepo) Update(ctx context.Context, d *domain.Device) error {
	query := `
	UPDATE devices SET
		name = ?,
		filtering_enabled = ?,
		parental_enabled = ?,
		safe_search_enabled = ?,
		blocked_services = ?,
		custom_blocked_domains = ?,
		custom_allowed_domains = ?,
		updated_at = ?
	WHERE id = ?;
	`
	servicesJSON, _ := json.Marshal(d.BlockedServices)
	blockedJSON, _ := json.Marshal(d.CustomBlockedDomains)
	allowedJSON, _ := json.Marshal(d.CustomAllowedDomains)

	d.UpdatedAt = time.Now().UTC()

	res, err := r.db.ExecContext(ctx, query,
		d.Name,
		d.FilteringEnabled,
		d.ParentalEnabled,
		d.SafeSearchEnabled,
		string(servicesJSON),
		string(blockedJSON),
		string(allowedJSON),
		d.UpdatedAt,
		d.ID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrDeviceNotFound
	}
	return nil
}

func (r *deviceRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM devices WHERE id = ?;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrDeviceNotFound
	}
	return nil
}

func (r *deviceRepo) CountByUserID(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM devices WHERE user_id = ?;`
	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDevice(s scannable) (*domain.Device, error) {
	var d domain.Device
	var servicesStr, blockedStr, allowedStr string

	err := s.Scan(
		&d.ID, &d.UserID, &d.Name, &d.ClientToken,
		&d.FilteringEnabled, &d.ParentalEnabled, &d.SafeSearchEnabled,
		&servicesStr, &blockedStr, &allowedStr,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrDeviceNotFound
	}
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(servicesStr), &d.BlockedServices)
	_ = json.Unmarshal([]byte(blockedStr), &d.CustomBlockedDomains)
	_ = json.Unmarshal([]byte(allowedStr), &d.CustomAllowedDomains)

	if d.BlockedServices == nil {
		d.BlockedServices = []string{}
	}
	if d.CustomBlockedDomains == nil {
		d.CustomBlockedDomains = []string{}
	}
	if d.CustomAllowedDomains == nil {
		d.CustomAllowedDomains = []string{}
	}

	return &d, nil
}

func scanDeviceRow(rows *sql.Rows) (*domain.Device, error) {
	return scanDevice(rows)
}
