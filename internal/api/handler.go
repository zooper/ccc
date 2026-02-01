package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/jonsson/ccc/internal/isp"
	"github.com/jonsson/ccc/internal/models"
	"github.com/jonsson/ccc/internal/storage"
)

const Version = "0.1.0"

// MetricsProvider interface for checking outage status and metrics
type MetricsProvider interface {
	HasAnyOutage() bool
	IsISPOutage(isp string) bool
	LastPingTime() time.Time
	PingInterval() time.Duration
	NextPingTime() time.Time
	PingCycleCount() int64
	StartTime() time.Time
}

// For backwards compatibility
type OutageChecker = MetricsProvider

// Handler holds dependencies for HTTP handlers
type Handler struct {
	db              *storage.DB
	dbPath          string
	classifier      *isp.Classifier
	metricsProvider MetricsProvider
	authRateLimiter *RateLimiter
}

// NewHandler creates a new API handler
func NewHandler(db *storage.DB, dbPath string, classifier *isp.Classifier) *Handler {
	return &Handler{
		db:         db,
		dbPath:     dbPath,
		classifier: classifier,
	}
}

// SetMetricsProvider sets the metrics provider (scheduler)
func (h *Handler) SetMetricsProvider(mp MetricsProvider) {
	h.metricsProvider = mp
}

// SetOutageChecker is an alias for SetMetricsProvider for backwards compatibility
func (h *Handler) SetOutageChecker(oc OutageChecker) {
	h.metricsProvider = oc
}

// SetAuthRateLimiter sets the rate limiter for auth endpoints
func (h *Handler) SetAuthRateLimiter(rl *RateLimiter) {
	h.authRateLimiter = rl
}

// Health handles GET /api/health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, models.HealthResponse{
		Status:  "ok",
		Version: Version,
	})
}

// Status handles GET /api/status
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	clientIP := GetClientIP(r)

	// Classify ISP for this client
	ispName, err := h.classifier.ClassifyISP(clientIP)
	if err != nil {
		log.Printf("ISP classification error for %s: %v", clientIP, err)
		ispName = "unknown"
	}

	// Check if client is registered
	endpoint, err := h.db.FindByIP(clientIP)
	if err != nil {
		log.Printf("Database error looking up %s: %v", clientIP, err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	response := models.StatusResponse{
		ISP:         ispName,
		Registered:  endpoint != nil,
		CanRegister: h.classifier.IsAllowed(ispName),
	}

	if endpoint != nil {
		response.EndpointID = &endpoint.ID
		response.EndpointStatus = endpoint.Status
		// Update last seen
		if err := h.db.UpdateLastSeen(endpoint.ID); err != nil {
			log.Printf("Failed to update last_seen for %s: %v", endpoint.ID, err)
		}
	}

	// Get ISP status if available
	if ispName != "unknown" {
		ispStatus, err := h.db.GetISPStatusByName(ispName)
		if err != nil {
			log.Printf("Failed to get ISP status for %s: %v", ispName, err)
		} else if ispStatus != nil {
			response.ISPStatus = ispStatus
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// Register handles POST /api/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	clientIP := GetClientIP(r)

	// Check if already registered
	existing, err := h.db.FindByIP(clientIP)
	if err != nil {
		log.Printf("Database error looking up %s: %v", clientIP, err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if existing != nil {
		// Already registered, update last_seen and return existing
		if err := h.db.UpdateLastSeen(existing.ID); err != nil {
			log.Printf("Failed to update last_seen for %s: %v", existing.ID, err)
		}
		writeJSON(w, http.StatusOK, models.RegisterResponse{
			EndpointID: existing.ID,
			ISP:        existing.ISP,
			Message:    "Already registered",
		})
		return
	}

	// Classify ISP
	ispName, err := h.classifier.ClassifyISP(clientIP)
	if err != nil {
		log.Printf("ISP classification error for %s: %v", clientIP, err)
		ispName = "unknown"
	}

	// Check if ISP is allowed to register
	if !h.classifier.IsAllowed(ispName) {
		log.Printf("Registration rejected for %s: ISP %s not allowed", clientIP, ispName)
		writeError(w, http.StatusForbidden, "Registration is only available for building residents")
		return
	}

	// Generate endpoint ID
	endpointID, err := generateEndpointID()
	if err != nil {
		log.Printf("Failed to generate endpoint ID: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to generate ID")
		return
	}

	// Create endpoint
	endpoint := &models.Endpoint{
		ID:        endpointID,
		IPv4:      clientIP,
		ISP:       ispName,
		Status:    "unknown",
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	if err := h.db.Create(endpoint); err != nil {
		log.Printf("Failed to create endpoint: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to register")
		return
	}

	log.Printf("Registered new endpoint: %s (ISP: %s)", endpointID, ispName)

	writeJSON(w, http.StatusCreated, models.RegisterResponse{
		EndpointID: endpointID,
		ISP:        ispName,
		Message:    "Successfully registered for monitoring",
	})
}

// Dashboard handles GET /api/dashboard
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetISPStats()
	if err != nil {
		log.Printf("Failed to get ISP stats: %v", err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Add ASN for each ISP (for icon lookup)
	for i := range stats {
		stats[i].ASN = h.classifier.GetASNForDisplay(stats[i].Name)
	}

	// Determine if there's a likely outage
	// First check the scheduler's hop-aware analysis
	likelyOutage := false
	if h.metricsProvider != nil {
		likelyOutage = h.metricsProvider.HasAnyOutage()
	}

	// Fallback: check if any ISP has >50% of endpoints down
	if !likelyOutage {
		for _, s := range stats {
			if s.TotalCount > 0 && float64(s.DownCount)/float64(s.TotalCount) > 0.5 {
				likelyOutage = true
				break
			}
		}
	}

	// Get last ping time from scheduler, fallback to now if not available
	lastUpdated := time.Now()
	if h.metricsProvider != nil {
		if pingTime := h.metricsProvider.LastPingTime(); !pingTime.IsZero() {
			lastUpdated = pingTime
		}
	}

	response := models.DashboardResponse{
		ISPs:         stats,
		LikelyOutage: likelyOutage,
		LastUpdated:  lastUpdated,
	}

	// Handle empty stats
	if response.ISPs == nil {
		response.ISPs = []models.ISPStatus{}
	}

	writeJSON(w, http.StatusOK, response)
}

// Events handles GET /api/events
func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	// Get recent events (last 24 hours)
	events, err := h.db.GetRecentEvents(24)
	if err != nil {
		log.Printf("Failed to get recent events: %v", err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if events == nil {
		events = []models.Event{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
	})
}

// generateEndpointID creates a random endpoint ID
func generateEndpointID() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("CCC-Endpoint-%s", hex.EncodeToString(bytes)), nil
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// Private IP ranges
var privateIPNets = []net.IPNet{
	// 10.0.0.0/8
	{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
	// 172.16.0.0/12
	{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
	// 192.168.0.0/16
	{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
	// 127.0.0.0/8 (loopback)
	{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
	// 169.254.0.0/16 (link-local)
	{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)},
	// 0.0.0.0/8
	{IP: net.IPv4(0, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
}

// isPrivateIP checks if an IP is in a private/reserved range
func isPrivateIP(ip net.IP) bool {
	for _, network := range privateIPNets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// AdminEndpoint is the admin view of an endpoint (includes IP)
type AdminEndpoint struct {
	ID           string    `json:"id"`
	IPv4         string    `json:"ipv4"`
	ISP          string    `json:"isp"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	LastSeen     time.Time `json:"last_seen"`
	LastOK       time.Time `json:"last_ok,omitempty"`
	MonitoredHop string    `json:"monitored_hop,omitempty"`
	HopNumber    int       `json:"hop_number,omitempty"`
	UseHop       bool      `json:"use_hop"`
}

// AdminListEndpoints handles GET /api/admin/endpoints
func (h *Handler) AdminListEndpoints(w http.ResponseWriter, r *http.Request) {
	endpoints, err := h.db.ListAll()
	if err != nil {
		log.Printf("Failed to list endpoints: %v", err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Convert to admin view
	result := make([]AdminEndpoint, len(endpoints))
	for i, e := range endpoints {
		result[i] = AdminEndpoint{
			ID:           e.ID,
			IPv4:         e.IPv4,
			ISP:          e.ISP,
			Status:       e.Status,
			CreatedAt:    e.CreatedAt,
			LastSeen:     e.LastSeen,
			LastOK:       e.LastOK,
			MonitoredHop: e.MonitoredHop,
			HopNumber:    e.HopNumber,
			UseHop:       e.UseHop,
		}
	}

	writeJSON(w, http.StatusOK, result)
}

// AdminAddEndpointRequest is the request body for adding an endpoint
type AdminAddEndpointRequest struct {
	IPv4 string `json:"ipv4"`
	ISP  string `json:"isp,omitempty"` // Optional, will auto-detect if empty
}

// AdminAddEndpoint handles POST /api/admin/endpoints
func (h *Handler) AdminAddEndpoint(w http.ResponseWriter, r *http.Request) {
	var req AdminAddEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.IPv4 == "" {
		writeError(w, http.StatusBadRequest, "ipv4 is required")
		return
	}

	// Validate IP address format
	parsedIP := net.ParseIP(req.IPv4)
	if parsedIP == nil {
		writeError(w, http.StatusBadRequest, "Invalid IPv4 address format")
		return
	}

	// Only allow IPv4
	if parsedIP.To4() == nil {
		writeError(w, http.StatusBadRequest, "Only IPv4 addresses are supported")
		return
	}

	// Block private/internal IPs for security
	if isPrivateIP(parsedIP) {
		writeError(w, http.StatusBadRequest, "Private/internal IP addresses are not allowed")
		return
	}

	// Check if already exists
	existing, err := h.db.FindByIP(req.IPv4)
	if err != nil {
		log.Printf("Database error looking up %s: %v", req.IPv4, err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if existing != nil {
		writeJSON(w, http.StatusOK, AdminEndpoint{
			ID:        existing.ID,
			IPv4:      existing.IPv4,
			ISP:       existing.ISP,
			Status:    existing.Status,
			CreatedAt: existing.CreatedAt,
			LastSeen:  existing.LastSeen,
			LastOK:    existing.LastOK,
		})
		return
	}

	// Classify ISP if not provided
	ispName := req.ISP
	if ispName == "" {
		ispName, err = h.classifier.ClassifyISP(req.IPv4)
		if err != nil {
			log.Printf("ISP classification error for %s: %v", req.IPv4, err)
			ispName = "unknown"
		}
	}

	// Generate endpoint ID
	endpointID, err := generateEndpointID()
	if err != nil {
		log.Printf("Failed to generate endpoint ID: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to generate ID")
		return
	}

	// Create endpoint
	endpoint := &models.Endpoint{
		ID:        endpointID,
		IPv4:      req.IPv4,
		ISP:       ispName,
		Status:    "unknown",
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	if err := h.db.Create(endpoint); err != nil {
		log.Printf("Failed to create endpoint: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to create endpoint")
		return
	}

	log.Printf("Admin added endpoint: %s (%s, ISP: %s)", endpointID, req.IPv4, ispName)

	writeJSON(w, http.StatusCreated, AdminEndpoint{
		ID:        endpoint.ID,
		IPv4:      endpoint.IPv4,
		ISP:       endpoint.ISP,
		Status:    endpoint.Status,
		CreatedAt: endpoint.CreatedAt,
		LastSeen:  endpoint.LastSeen,
	})
}

// AdminDeleteEndpoint handles DELETE /api/admin/endpoints/{id}
func (h *Handler) AdminDeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Endpoint ID is required")
		return
	}

	deleted, err := h.db.DeleteByID(id)
	if err != nil {
		log.Printf("Failed to delete endpoint %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if !deleted {
		writeError(w, http.StatusNotFound, "Endpoint not found")
		return
	}

	log.Printf("Admin deleted endpoint: %s", id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Endpoint deleted"})
}

// AdminMetrics handles GET /api/admin/metrics
func (h *Handler) AdminMetrics(w http.ResponseWriter, r *http.Request) {
	// Get endpoint metrics
	total, up, down, unknown, direct, hopMonitored, err := h.db.GetEndpointMetrics()
	if err != nil {
		log.Printf("Failed to get endpoint metrics: %v", err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Get ISP metrics
	ispStats, err := h.db.GetISPMetrics()
	if err != nil {
		log.Printf("Failed to get ISP metrics: %v", err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Get shared hop count
	sharedHops, err := h.db.GetSharedHopCount()
	if err != nil {
		log.Printf("Failed to get shared hop count: %v", err)
		sharedHops = 0
	}

	// Get uptime history (last 24 hours)
	history, err := h.db.GetUptimeHistory(24 * time.Hour)
	if err != nil {
		log.Printf("Failed to get uptime history: %v", err)
		history = []models.UptimePoint{}
	}

	// Get database size
	dbSize, err := h.db.GetDatabaseSize(h.dbPath)
	if err != nil {
		log.Printf("Failed to get database size: %v", err)
		dbSize = 0
	}

	// Calculate overall uptime percentage
	var overallUptimePct float64
	if total > 0 {
		overallUptimePct = float64(up) / float64(total) * 100
	}

	// Get scheduler metrics
	var lastPingTime, nextPingTime, serverStartTime time.Time
	var pingInterval string
	var totalPingCycles int64

	if h.metricsProvider != nil {
		lastPingTime = h.metricsProvider.LastPingTime()
		nextPingTime = h.metricsProvider.NextPingTime()
		pingInterval = h.metricsProvider.PingInterval().String()
		totalPingCycles = h.metricsProvider.PingCycleCount()
		serverStartTime = h.metricsProvider.StartTime()
	}

	// Calculate server uptime
	var serverUptime string
	if !serverStartTime.IsZero() {
		uptime := time.Since(serverStartTime)
		hours := int(uptime.Hours())
		minutes := int(uptime.Minutes()) % 60
		if hours > 24 {
			days := hours / 24
			hours = hours % 24
			serverUptime = fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
		} else {
			serverUptime = fmt.Sprintf("%dh %dm", hours, minutes)
		}
	}

	metrics := models.AdminMetrics{
		TotalEndpoints:   total,
		EndpointsUp:      up,
		EndpointsDown:    down,
		EndpointsUnknown: unknown,
		OverallUptimePct: overallUptimePct,
		ISPStats:         ispStats,
		LastPingTime:     lastPingTime,
		PingInterval:     pingInterval,
		NextPingTime:     nextPingTime,
		TotalPingCycles:  totalPingCycles,
		DirectMonitored:  direct,
		HopMonitored:     hopMonitored,
		SharedHops:       sharedHops,
		ServerStartTime:  serverStartTime,
		ServerUptime:     serverUptime,
		Version:          Version,
		DatabaseSize:     dbSize,
		DatabasePath:     h.dbPath,
		UptimeHistory:    history,
	}

	// Handle nil slices for JSON
	if metrics.ISPStats == nil {
		metrics.ISPStats = []models.ISPMetrics{}
	}
	if metrics.UptimeHistory == nil {
		metrics.UptimeHistory = []models.UptimePoint{}
	}

	writeJSON(w, http.StatusOK, metrics)
}
