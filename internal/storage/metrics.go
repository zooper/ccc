package storage

import (
	"fmt"
	"os"
	"time"

	"github.com/jonsson/ccc/internal/models"
)

// RecordUptimeSnapshot records the current uptime status for historical tracking
func (db *DB) RecordUptimeSnapshot(total, up, down int) error {
	_, err := db.conn.Exec(`
		INSERT INTO uptime_history (timestamp, total_endpoints, endpoints_up, endpoints_down)
		VALUES (?, ?, ?, ?)
	`, time.Now(), total, up, down)
	if err != nil {
		return fmt.Errorf("failed to record uptime snapshot: %w", err)
	}
	return nil
}

// CleanupOldHistory removes history older than the specified duration
func (db *DB) CleanupOldHistory(maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge)
	result, err := db.conn.Exec(`DELETE FROM uptime_history WHERE timestamp < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old history: %w", err)
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// GetUptimeHistory returns uptime history for the specified duration
func (db *DB) GetUptimeHistory(since time.Duration) ([]models.UptimePoint, error) {
	cutoff := time.Now().Add(-since)
	rows, err := db.conn.Query(`
		SELECT timestamp, endpoints_up, endpoints_down
		FROM uptime_history
		WHERE timestamp > ?
		ORDER BY timestamp ASC
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to get uptime history: %w", err)
	}
	defer rows.Close()

	var history []models.UptimePoint
	for rows.Next() {
		var p models.UptimePoint
		var ts string
		if err := rows.Scan(&ts, &p.Up, &p.Down); err != nil {
			return nil, fmt.Errorf("failed to scan history row: %w", err)
		}
		p.Timestamp = parseTime(ts)
		total := p.Up + p.Down
		if total > 0 {
			p.UptimePct = float64(p.Up) / float64(total) * 100
		}
		history = append(history, p)
	}
	return history, nil
}

// GetEndpointMetrics returns aggregated endpoint metrics
func (db *DB) GetEndpointMetrics() (total, up, down, unknown, direct, hopMonitored int, err error) {
	row := db.conn.QueryRow(`
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END), 0) as up,
			COALESCE(SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END), 0) as down,
			COALESCE(SUM(CASE WHEN status = 'unknown' THEN 1 ELSE 0 END), 0) as unknown,
			COALESCE(SUM(CASE WHEN use_hop = 0 OR use_hop IS NULL THEN 1 ELSE 0 END), 0) as direct,
			COALESCE(SUM(CASE WHEN use_hop = 1 THEN 1 ELSE 0 END), 0) as hop_monitored
		FROM endpoints
	`)
	err = row.Scan(&total, &up, &down, &unknown, &direct, &hopMonitored)
	if err != nil {
		err = fmt.Errorf("failed to get endpoint metrics: %w", err)
	}
	return
}

// GetSharedHopCount returns the number of unique hops shared by multiple endpoints
func (db *DB) GetSharedHopCount() (int, error) {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(DISTINCT monitored_hop)
		FROM endpoints
		WHERE monitored_hop IS NOT NULL AND monitored_hop != ''
		AND monitored_hop IN (
			SELECT monitored_hop FROM endpoints
			WHERE monitored_hop IS NOT NULL AND monitored_hop != ''
			GROUP BY monitored_hop HAVING COUNT(*) > 1
		)
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get shared hop count: %w", err)
	}
	return count, nil
}

// GetISPMetrics returns detailed metrics per ISP
func (db *DB) GetISPMetrics() ([]models.ISPMetrics, error) {
	rows, err := db.conn.Query(`
		SELECT
			isp,
			COUNT(*) as total,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up,
			SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END) as down,
			SUM(CASE WHEN status = 'unknown' THEN 1 ELSE 0 END) as unknown
		FROM endpoints
		GROUP BY isp
		ORDER BY total DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get ISP metrics: %w", err)
	}
	defer rows.Close()

	var metrics []models.ISPMetrics
	for rows.Next() {
		var m models.ISPMetrics
		if err := rows.Scan(&m.Name, &m.Total, &m.Up, &m.Down, &m.Unknown); err != nil {
			return nil, fmt.Errorf("failed to scan ISP metrics: %w", err)
		}
		if m.Total > 0 {
			m.UptimePct = float64(m.Up) / float64(m.Total) * 100
		}
		// Mark as likely outage if >50% down
		m.LikelyOutage = m.Total > 0 && float64(m.Down)/float64(m.Total) > 0.5
		metrics = append(metrics, m)
	}
	return metrics, nil
}

// GetDatabaseSize returns the size of the database file in bytes
func (db *DB) GetDatabaseSize(dbPath string) (int64, error) {
	info, err := os.Stat(dbPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetHistoryCount returns the number of history records
func (db *DB) GetHistoryCount() (int64, error) {
	var count int64
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM uptime_history`).Scan(&count)
	return count, err
}

// RecordEvent adds a new event to the events table
func (db *DB) RecordEvent(eventType, isp, endpointID, message string) error {
	_, err := db.conn.Exec(`
		INSERT INTO events (timestamp, event_type, isp, endpoint_id, message)
		VALUES (?, ?, ?, ?, ?)
	`, time.Now(), eventType, isp, endpointID, message)
	if err != nil {
		return fmt.Errorf("failed to record event: %w", err)
	}
	return nil
}

// GetRecentEvents returns recent events (last N hours)
func (db *DB) GetRecentEvents(hours int) ([]models.Event, error) {
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	rows, err := db.conn.Query(`
		SELECT id, timestamp, event_type, COALESCE(isp, ''), COALESCE(endpoint_id, ''), message
		FROM events
		WHERE timestamp > ?
		ORDER BY timestamp DESC
		LIMIT 50
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		var ts string
		if err := rows.Scan(&e.ID, &ts, &e.EventType, &e.ISP, &e.EndpointID, &e.Message); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		e.Timestamp = parseTime(ts)
		events = append(events, e)
	}
	return events, nil
}

// CleanupOldEvents removes events older than the specified duration
func (db *DB) CleanupOldEvents(maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge)
	result, err := db.conn.Exec(`DELETE FROM events WHERE timestamp < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old events: %w", err)
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}
