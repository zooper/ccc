package storage

import "log"

const schema = `
CREATE TABLE IF NOT EXISTS endpoints (
    id TEXT PRIMARY KEY,
    ipv4 TEXT NOT NULL,
    ip_hash TEXT NOT NULL UNIQUE,
    isp TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unknown',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_ok DATETIME,
    monitored_hop TEXT,
    hop_number INTEGER DEFAULT 0,
    use_hop INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS uptime_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    total_endpoints INTEGER NOT NULL,
    endpoints_up INTEGER NOT NULL,
    endpoints_down INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type TEXT NOT NULL,
    isp TEXT,
    endpoint_id TEXT,
    message TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_endpoints_ip_hash ON endpoints(ip_hash);
CREATE INDEX IF NOT EXISTS idx_endpoints_isp ON endpoints(isp);
CREATE INDEX IF NOT EXISTS idx_endpoints_status ON endpoints(status);
CREATE INDEX IF NOT EXISTS idx_endpoints_last_seen ON endpoints(last_seen);
CREATE INDEX IF NOT EXISTS idx_endpoints_monitored_hop ON endpoints(monitored_hop);
CREATE INDEX IF NOT EXISTS idx_uptime_history_timestamp ON uptime_history(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
`

// Migration to add hop columns to existing databases
const migrationAddHopColumns = `
ALTER TABLE endpoints ADD COLUMN monitored_hop TEXT;
ALTER TABLE endpoints ADD COLUMN hop_number INTEGER DEFAULT 0;
ALTER TABLE endpoints ADD COLUMN use_hop INTEGER DEFAULT 0;
`

func (db *DB) migrate() error {
	_, err := db.conn.Exec(schema)
	if err != nil {
		return err
	}

	// Try to add hop columns for existing databases (will fail if already exist)
	db.conn.Exec("ALTER TABLE endpoints ADD COLUMN monitored_hop TEXT")
	db.conn.Exec("ALTER TABLE endpoints ADD COLUMN hop_number INTEGER DEFAULT 0")
	db.conn.Exec("ALTER TABLE endpoints ADD COLUMN use_hop INTEGER DEFAULT 0")

	// Create index if it doesn't exist
	db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_endpoints_monitored_hop ON endpoints(monitored_hop)")

	// Add events table for existing databases (will fail if already exists via schema)
	db.conn.Exec(`CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		event_type TEXT NOT NULL,
		isp TEXT,
		endpoint_id TEXT,
		message TEXT NOT NULL
	)`)
	db.conn.Exec("CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp)")

	log.Println("Database migrations completed")
	return nil
}
