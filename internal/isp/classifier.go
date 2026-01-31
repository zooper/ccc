package isp

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// ISPConfig represents the configuration for a single ISP
type ISPConfig struct {
	Display string `json:"display"`
	Allowed bool   `json:"allowed"`
}

// Classifier handles ISP classification via ASN lookups
type Classifier struct {
	cache        map[string]cacheEntry
	cacheOrder   []string // Track insertion order for LRU-ish eviction
	cacheMu      sync.RWMutex
	cacheTTL     time.Duration
	maxCacheSize int
	asnConfig    map[int]ISPConfig // ASN -> config mapping
}

type cacheEntry struct {
	isp       string
	allowed   bool
	expiresAt time.Time
}

// NewClassifier creates a new ISP classifier
func NewClassifier() *Classifier {
	return &Classifier{
		cache:        make(map[string]cacheEntry),
		cacheOrder:   make([]string, 0),
		cacheTTL:     24 * time.Hour,
		maxCacheSize: 10000,
		asnConfig:    make(map[int]ISPConfig),
	}
}

// LoadConfig loads ISP configuration from a JSON file
func (c *Classifier) LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read ISP config: %w", err)
	}

	// Parse JSON with string keys (ASN numbers as strings)
	var rawConfig map[string]ISPConfig
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed to parse ISP config: %w", err)
	}

	// Convert string keys to int
	c.asnConfig = make(map[int]ISPConfig)
	for asnStr, config := range rawConfig {
		var asn int
		if _, err := fmt.Sscanf(asnStr, "%d", &asn); err != nil {
			log.Printf("Warning: invalid ASN in config: %s", asnStr)
			continue
		}
		c.asnConfig[asn] = config
	}

	log.Printf("Loaded ISP config: %d ASN mappings", len(c.asnConfig))
	return nil
}

// ClassifyISP returns the ISP display name for an IP address
func (c *Classifier) ClassifyISP(ip string) (string, error) {
	// Check cache first
	c.cacheMu.RLock()
	if entry, ok := c.cache[ip]; ok && time.Now().Before(entry.expiresAt) {
		c.cacheMu.RUnlock()
		return entry.isp, nil
	}
	c.cacheMu.RUnlock()

	// Perform ASN lookup
	asn, _, err := c.LookupASN(ip)
	if err != nil {
		return "Unknown", err
	}

	if asn == 0 {
		return "Unknown", nil
	}

	// Look up ASN in config
	var ispName string
	if config, ok := c.asnConfig[asn]; ok {
		ispName = config.Display
	} else {
		// Fallback: get org name from ASN info
		_, org, err := c.LookupASNInfo(asn)
		if err != nil || org == "" {
			ispName = "Unknown"
		} else {
			// Use a cleaned-up version of the org name
			ispName = cleanOrgName(org)
		}
	}

	// Cache the result with size limit
	c.cacheMu.Lock()

	// If cache is at max size, remove oldest entries
	for len(c.cache) >= c.maxCacheSize && len(c.cacheOrder) > 0 {
		oldest := c.cacheOrder[0]
		c.cacheOrder = c.cacheOrder[1:]
		delete(c.cache, oldest)
	}

	c.cache[ip] = cacheEntry{
		isp:       ispName,
		expiresAt: time.Now().Add(c.cacheTTL),
	}
	c.cacheOrder = append(c.cacheOrder, ip)
	c.cacheMu.Unlock()

	return ispName, nil
}

// IsAllowed checks if an ISP (by display name) is allowed to register
func (c *Classifier) IsAllowed(ispDisplay string) bool {
	for _, config := range c.asnConfig {
		if config.Display == ispDisplay {
			return config.Allowed
		}
	}
	return false
}

// IsASNAllowed checks if a specific ASN is allowed to register
func (c *Classifier) IsASNAllowed(asn int) bool {
	if config, ok := c.asnConfig[asn]; ok {
		return config.Allowed
	}
	return false
}

// GetAllowedISPs returns a list of all allowed ISP display names
func (c *Classifier) GetAllowedISPs() []string {
	seen := make(map[string]bool)
	var allowed []string
	for _, config := range c.asnConfig {
		if config.Allowed && !seen[config.Display] {
			seen[config.Display] = true
			allowed = append(allowed, config.Display)
		}
	}
	return allowed
}

// GetASNForDisplay returns the first ASN for a given display name
func (c *Classifier) GetASNForDisplay(display string) int {
	for asn, config := range c.asnConfig {
		if config.Display == display {
			return asn
		}
	}
	return 0
}

// cleanOrgName extracts a cleaner name from ASN org string
// e.g., "COMCAST-7922 - Comcast Cable Communications, Inc., US" -> "Comcast Cable Communications"
func cleanOrgName(org string) string {
	// Try to get the part after " - "
	if idx := strings.Index(org, " - "); idx > 0 {
		org = org[idx+3:]
	}
	// Remove trailing ", US" or similar country codes
	if idx := strings.LastIndex(org, ", "); idx > 0 && len(org)-idx <= 5 {
		org = org[:idx]
	}
	// Remove ", Inc." or ", LLC" etc.
	for _, suffix := range []string{", Inc.", ", LLC", ", Ltd.", ", Corp."} {
		org = strings.TrimSuffix(org, suffix)
	}
	return strings.TrimSpace(org)
}

// LookupASN queries Team Cymru DNS for ASN information
// Query format: reverse IP octets + ".origin.asn.cymru.com"
// Response format: "ASN | CIDR | CC | Registry | Date"
func (c *Classifier) LookupASN(ip string) (asn int, cidr string, err error) {
	// Parse and validate IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return 0, "", fmt.Errorf("invalid IP address: %s", ip)
	}

	// Only support IPv4 for now
	ipv4 := parsedIP.To4()
	if ipv4 == nil {
		return 0, "", fmt.Errorf("IPv6 not supported: %s", ip)
	}

	// Reverse the IP octets
	reversed := fmt.Sprintf("%d.%d.%d.%d", ipv4[3], ipv4[2], ipv4[1], ipv4[0])
	query := reversed + ".origin.asn.cymru.com"

	// Perform DNS TXT lookup
	records, err := net.LookupTXT(query)
	if err != nil {
		return 0, "", fmt.Errorf("ASN lookup failed: %w", err)
	}

	if len(records) == 0 {
		return 0, "", nil
	}

	// Parse response: "7922 | 1.2.3.0/24 | US | arin | 1997-12-01"
	parts := strings.Split(records[0], "|")
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("unexpected ASN response format: %s", records[0])
	}

	// Parse ASN (might have multiple ASNs, take the first)
	asnStr := strings.TrimSpace(parts[0])
	asnParts := strings.Fields(asnStr)
	if len(asnParts) > 0 {
		fmt.Sscanf(asnParts[0], "%d", &asn)
	}

	cidr = strings.TrimSpace(parts[1])
	return asn, cidr, nil
}

// LookupASNInfo queries Team Cymru DNS for ASN details
// Query format: "AS" + ASN + ".asn.cymru.com"
// Response format: "ASN | CC | Registry | Date | Name"
func (c *Classifier) LookupASNInfo(asn int) (name string, org string, err error) {
	query := fmt.Sprintf("AS%d.asn.cymru.com", asn)

	records, err := net.LookupTXT(query)
	if err != nil {
		return "", "", fmt.Errorf("ASN info lookup failed: %w", err)
	}

	if len(records) == 0 {
		return "", "", nil
	}

	// Parse response: "7922 | US | arin | 1997-12-01 | COMCAST-7922 - Comcast Cable Communications, Inc., US"
	parts := strings.Split(records[0], "|")
	if len(parts) < 5 {
		return "", "", fmt.Errorf("unexpected ASN info format: %s", records[0])
	}

	org = strings.TrimSpace(parts[4])
	// Extract name (before the dash if present)
	if idx := strings.Index(org, " - "); idx > 0 {
		name = org[:idx]
	} else {
		name = org
	}

	return name, org, nil
}

// ClearCache removes all cached entries
func (c *Classifier) ClearCache() {
	c.cacheMu.Lock()
	c.cache = make(map[string]cacheEntry)
	c.cacheOrder = make([]string, 0)
	c.cacheMu.Unlock()
}

// CacheSize returns the number of cached entries
func (c *Classifier) CacheSize() int {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	return len(c.cache)
}
