package sqlite

func (db *DB) runMigrations() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS devices (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		client_token TEXT NOT NULL UNIQUE,
		filtering_enabled BOOLEAN NOT NULL DEFAULT 1,
		parental_enabled BOOLEAN NOT NULL DEFAULT 0,
		safe_search_enabled BOOLEAN NOT NULL DEFAULT 0,
		blocked_services TEXT NOT NULL DEFAULT '[]',
		custom_blocked_domains TEXT NOT NULL DEFAULT '[]',
		custom_allowed_domains TEXT NOT NULL DEFAULT '[]',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_client_token ON devices(client_token);
	`

	_, err := db.Exec(schema)
	return err
}
