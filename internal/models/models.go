package models

import "time"

// Endpoint represents a monitored IP endpoint
type Endpoint struct {
	ID           string    `json:"id"`            // e.g., "CCC-Endpoint-0123"
	IPv4         string    `json:"-"`             // Stored for monitoring, not exposed in API
	IPHash       string    `json:"ip_hash"`       // SHA256 hash for lookup
	ISP          string    `json:"isp"`           // "starry", "comcast", "unknown"
	Status       string    `json:"status"`        // "up", "down", "unknown"
	CreatedAt    time.Time `json:"created_at"`
	LastSeen     time.Time `json:"last_seen"`
	LastOK       time.Time `json:"last_ok"`
	MonitoredHop string    `json:"-"`             // IP of hop being monitored (if different from IPv4)
	HopNumber    int       `json:"hop_number"`    // TTL/hop number of monitored hop (0 = direct)
	UseHop       bool      `json:"use_hop"`       // True if monitoring a hop instead of direct IP
}

// ISPStatus represents aggregated status for an ISP
type ISPStatus struct {
	Name        string    `json:"name"`
	ASN         int       `json:"asn,omitempty"`
	TotalCount  int       `json:"total"`
	UpCount     int       `json:"up"`
	DownCount   int       `json:"down"`
	LastUpdated time.Time `json:"last_updated"`
}

// StatusResponse is returned by GET /api/status
type StatusResponse struct {
	ISP            string     `json:"isp"`
	Registered     bool       `json:"registered"`
	CanRegister    bool       `json:"can_register"`              // True if ISP is allowed to register
	EndpointID     *string    `json:"endpoint_id"`
	EndpointStatus string     `json:"endpoint_status,omitempty"` // "up", "down", "unreachable", "unknown"
	ISPStatus      *ISPStatus `json:"isp_status,omitempty"`
}

// RegisterResponse is returned by POST /api/register
type RegisterResponse struct {
	EndpointID string `json:"endpoint_id"`
	ISP        string `json:"isp"`
	Message    string `json:"message"`
}

// DashboardResponse is returned by GET /api/dashboard
type DashboardResponse struct {
	ISPs         []ISPStatus `json:"isps"`
	LikelyOutage bool        `json:"likely_outage"`
	LastUpdated  time.Time   `json:"last_updated"`
}

// HealthResponse is returned by GET /api/health
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// AdminMetrics contains comprehensive system metrics
type AdminMetrics struct {
	// Overview
	TotalEndpoints   int     `json:"total_endpoints"`
	EndpointsUp      int     `json:"endpoints_up"`
	EndpointsDown    int     `json:"endpoints_down"`
	EndpointsUnknown int     `json:"endpoints_unknown"`
	OverallUptimePct float64 `json:"overall_uptime_pct"`

	// Per-ISP breakdown
	ISPStats []ISPMetrics `json:"isp_stats"`

	// Monitoring stats
	LastPingTime     time.Time `json:"last_ping_time"`
	PingInterval     string    `json:"ping_interval"`
	NextPingTime     time.Time `json:"next_ping_time"`
	TotalPingCycles  int64     `json:"total_ping_cycles"`

	// Endpoint details
	DirectMonitored  int `json:"direct_monitored"`   // Endpoints monitored directly
	HopMonitored     int `json:"hop_monitored"`      // Endpoints monitored via hop
	SharedHops       int `json:"shared_hops"`        // Number of shared hops

	// System info
	ServerStartTime  time.Time `json:"server_start_time"`
	ServerUptime     string    `json:"server_uptime"`
	Version          string    `json:"version"`
	DatabaseSize     int64     `json:"database_size_bytes"`
	DatabasePath     string    `json:"database_path"`

	// Historical (last 24h)
	UptimeHistory    []UptimePoint `json:"uptime_history"`
}

// ISPMetrics contains per-ISP metrics
type ISPMetrics struct {
	Name         string  `json:"name"`
	Total        int     `json:"total"`
	Up           int     `json:"up"`
	Down         int     `json:"down"`
	Unknown      int     `json:"unknown"`
	UptimePct    float64 `json:"uptime_pct"`
	LikelyOutage bool    `json:"likely_outage"`
}

// UptimePoint is a historical uptime data point
type UptimePoint struct {
	Timestamp time.Time `json:"timestamp"`
	UptimePct float64   `json:"uptime_pct"`
	Up        int       `json:"up"`
	Down      int       `json:"down"`
}

// Event represents a status change or notable occurrence
type Event struct {
	ID         int64     `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	EventType  string    `json:"event_type"`  // "down", "up", "outage", "recovery", "registered"
	ISP        string    `json:"isp,omitempty"`
	EndpointID string    `json:"endpoint_id,omitempty"`
	Message    string    `json:"message"`
}
