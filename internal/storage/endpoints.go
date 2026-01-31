package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jonsson/ccc/internal/models"
)

// HashIP creates a SHA256 hash of an IP address
func HashIP(ip string) string {
	h := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(h[:])
}

// FindByIPHash finds an endpoint by its IP hash
func (db *DB) FindByIPHash(ipHash string) (*models.Endpoint, error) {
	row := db.conn.QueryRow(`
		SELECT id, ipv4, ip_hash, isp, status, created_at, last_seen, last_ok,
		       COALESCE(monitored_hop, ''), COALESCE(hop_number, 0), COALESCE(use_hop, 0)
		FROM endpoints WHERE ip_hash = ?
	`, ipHash)

	var e models.Endpoint
	var lastOK sql.NullTime
	var useHopInt int
	err := row.Scan(&e.ID, &e.IPv4, &e.IPHash, &e.ISP, &e.Status, &e.CreatedAt, &e.LastSeen, &lastOK,
		&e.MonitoredHop, &e.HopNumber, &useHopInt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find endpoint: %w", err)
	}
	if lastOK.Valid {
		e.LastOK = lastOK.Time
	}
	e.UseHop = useHopInt != 0
	return &e, nil
}

// FindByIP finds an endpoint by its IP address
func (db *DB) FindByIP(ip string) (*models.Endpoint, error) {
	return db.FindByIPHash(HashIP(ip))
}

// Create inserts a new endpoint
func (db *DB) Create(e *models.Endpoint) error {
	e.IPHash = HashIP(e.IPv4)
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	if e.LastSeen.IsZero() {
		e.LastSeen = time.Now()
	}
	if e.Status == "" {
		e.Status = "unknown"
	}

	useHopInt := 0
	if e.UseHop {
		useHopInt = 1
	}

	_, err := db.conn.Exec(`
		INSERT INTO endpoints (id, ipv4, ip_hash, isp, status, created_at, last_seen, last_ok, monitored_hop, hop_number, use_hop)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, e.ID, e.IPv4, e.IPHash, e.ISP, e.Status, e.CreatedAt, e.LastSeen,
		sql.NullTime{Time: e.LastOK, Valid: !e.LastOK.IsZero()},
		sql.NullString{String: e.MonitoredHop, Valid: e.MonitoredHop != ""},
		e.HopNumber, useHopInt)
	if err != nil {
		return fmt.Errorf("failed to create endpoint: %w", err)
	}
	return nil
}

// UpdateStatus updates the status of an endpoint
func (db *DB) UpdateStatus(id, status string, lastOK time.Time) error {
	var lastOKVal sql.NullTime
	if !lastOK.IsZero() {
		lastOKVal = sql.NullTime{Time: lastOK, Valid: true}
	}

	_, err := db.conn.Exec(`
		UPDATE endpoints SET status = ?, last_ok = COALESCE(?, last_ok)
		WHERE id = ?
	`, status, lastOKVal, id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	return nil
}

// UpdateLastSeen updates the last_seen timestamp
func (db *DB) UpdateLastSeen(id string) error {
	_, err := db.conn.Exec(`
		UPDATE endpoints SET last_seen = CURRENT_TIMESTAMP WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("failed to update last_seen: %w", err)
	}
	return nil
}

// ListByISP returns all endpoints for a given ISP
func (db *DB) ListByISP(isp string) ([]models.Endpoint, error) {
	rows, err := db.conn.Query(`
		SELECT id, ipv4, ip_hash, isp, status, created_at, last_seen, last_ok,
		       COALESCE(monitored_hop, ''), COALESCE(hop_number, 0), COALESCE(use_hop, 0)
		FROM endpoints WHERE isp = ?
	`, isp)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}
	defer rows.Close()

	return scanEndpoints(rows)
}

// ListAll returns all endpoints
func (db *DB) ListAll() ([]models.Endpoint, error) {
	rows, err := db.conn.Query(`
		SELECT id, ipv4, ip_hash, isp, status, created_at, last_seen, last_ok,
		       COALESCE(monitored_hop, ''), COALESCE(hop_number, 0), COALESCE(use_hop, 0)
		FROM endpoints
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}
	defer rows.Close()

	return scanEndpoints(rows)
}

// scanEndpoints is a helper to scan endpoint rows
func scanEndpoints(rows *sql.Rows) ([]models.Endpoint, error) {
	var endpoints []models.Endpoint
	for rows.Next() {
		var e models.Endpoint
		var lastOK sql.NullTime
		var useHopInt int
		if err := rows.Scan(&e.ID, &e.IPv4, &e.IPHash, &e.ISP, &e.Status, &e.CreatedAt, &e.LastSeen, &lastOK,
			&e.MonitoredHop, &e.HopNumber, &useHopInt); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint: %w", err)
		}
		if lastOK.Valid {
			e.LastOK = lastOK.Time
		}
		e.UseHop = useHopInt != 0
		endpoints = append(endpoints, e)
	}
	return endpoints, nil
}

// UpdateMonitoredHop updates the hop being monitored for an endpoint
func (db *DB) UpdateMonitoredHop(id, hopIP string, hopNumber int) error {
	useHop := 0
	if hopIP != "" {
		useHop = 1
	}

	_, err := db.conn.Exec(`
		UPDATE endpoints SET monitored_hop = ?, hop_number = ?, use_hop = ?
		WHERE id = ?
	`, sql.NullString{String: hopIP, Valid: hopIP != ""}, hopNumber, useHop, id)
	if err != nil {
		return fmt.Errorf("failed to update monitored hop: %w", err)
	}
	return nil
}

// GetEndpointsByMonitoredHop returns all endpoints monitoring the same hop
func (db *DB) GetEndpointsByMonitoredHop(hopIP string) ([]models.Endpoint, error) {
	rows, err := db.conn.Query(`
		SELECT id, ipv4, ip_hash, isp, status, created_at, last_seen, last_ok,
		       COALESCE(monitored_hop, ''), COALESCE(hop_number, 0), COALESCE(use_hop, 0)
		FROM endpoints WHERE monitored_hop = ?
	`, hopIP)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints by hop: %w", err)
	}
	defer rows.Close()

	return scanEndpoints(rows)
}

// DeleteExpired removes endpoints not seen in the specified number of days
func (db *DB) DeleteExpired(maxAgeDays int) (int, error) {
	result, err := db.conn.Exec(`
		DELETE FROM endpoints WHERE last_seen < datetime('now', '-' || ? || ' days')
	`, maxAgeDays)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired endpoints: %w", err)
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// DeleteByID removes an endpoint by its ID
func (db *DB) DeleteByID(id string) (bool, error) {
	result, err := db.conn.Exec(`DELETE FROM endpoints WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("failed to delete endpoint: %w", err)
	}
	count, _ := result.RowsAffected()
	return count > 0, nil
}

// GetISPStats returns aggregated statistics by ISP
func (db *DB) GetISPStats() ([]models.ISPStatus, error) {
	rows, err := db.conn.Query(`
		SELECT
			isp,
			COUNT(*) as total,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up_count,
			SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END) as down_count,
			MAX(last_seen) as last_updated
		FROM endpoints
		GROUP BY isp
		ORDER BY total DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get ISP stats: %w", err)
	}
	defer rows.Close()

	var stats []models.ISPStatus
	for rows.Next() {
		var s models.ISPStatus
		var lastUpdatedStr string
		if err := rows.Scan(&s.Name, &s.TotalCount, &s.UpCount, &s.DownCount, &lastUpdatedStr); err != nil {
			return nil, fmt.Errorf("failed to scan ISP stats: %w", err)
		}
		s.LastUpdated = parseTime(lastUpdatedStr)
		stats = append(stats, s)
	}
	return stats, nil
}

// GetISPStatusByName returns stats for a specific ISP
func (db *DB) GetISPStatusByName(isp string) (*models.ISPStatus, error) {
	row := db.conn.QueryRow(`
		SELECT
			isp,
			COUNT(*) as total,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up_count,
			SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END) as down_count,
			MAX(last_seen) as last_updated
		FROM endpoints
		WHERE isp = ?
		GROUP BY isp
	`, isp)

	var s models.ISPStatus
	var lastUpdatedStr string
	err := row.Scan(&s.Name, &s.TotalCount, &s.UpCount, &s.DownCount, &lastUpdatedStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ISP status: %w", err)
	}
	s.LastUpdated = parseTime(lastUpdatedStr)
	return &s, nil
}

// parseTime attempts to parse a time string in various formats
func parseTime(s string) time.Time {
	formats := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
