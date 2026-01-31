package isp

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// Classifier handles ISP classification via ASN lookups
type Classifier struct {
	cache       map[string]cacheEntry
	cacheOrder  []string // Track insertion order for LRU-ish eviction
	cacheMu     sync.RWMutex
	cacheTTL    time.Duration
	maxCacheSize int
}

type cacheEntry struct {
	isp       string
	expiresAt time.Time
}

// NewClassifier creates a new ISP classifier
func NewClassifier() *Classifier {
	return &Classifier{
		cache:       make(map[string]cacheEntry),
		cacheOrder:  make([]string, 0),
		cacheTTL:    24 * time.Hour,
		maxCacheSize: 10000, // Limit cache to 10k entries
	}
}

// ClassifyISP returns the normalized ISP name for an IP address
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
		return "unknown", err
	}

	if asn == 0 {
		return "unknown", nil
	}

	// Get ASN info
	_, org, err := c.LookupASNInfo(asn)
	if err != nil {
		return "unknown", err
	}

	// Normalize ISP name
	isp := c.normalizeISP(org)

	// Cache the result with size limit
	c.cacheMu.Lock()

	// If cache is at max size, remove oldest entries
	for len(c.cache) >= c.maxCacheSize && len(c.cacheOrder) > 0 {
		oldest := c.cacheOrder[0]
		c.cacheOrder = c.cacheOrder[1:]
		delete(c.cache, oldest)
	}

	c.cache[ip] = cacheEntry{
		isp:       isp,
		expiresAt: time.Now().Add(c.cacheTTL),
	}
	c.cacheOrder = append(c.cacheOrder, ip)
	c.cacheMu.Unlock()

	return isp, nil
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

// normalizeISP maps ASN organization names to normalized ISP identifiers
func (c *Classifier) normalizeISP(org string) string {
	orgUpper := strings.ToUpper(org)

	// Known ISP mappings
	ispMappings := map[string]string{
		"COMCAST":   "comcast",
		"XFINITY":   "comcast",
		"STARRY":    "starry",
		"VERIZON":   "verizon",
		"FIOS":      "verizon",
		"ATT":       "att",
		"AT&T":      "att",
		"SPECTRUM":  "spectrum",
		"CHARTER":   "spectrum",
		"RCNCF":     "rcn",
		"RCN":       "rcn",
		"OPTIMUM":   "optimum",
		"CABLEVISION": "optimum",
		"COX":       "cox",
		"GOOGLE":    "google-fiber",
		"T-MOBILE":  "tmobile",
		"TMOBILE":   "tmobile",
	}

	for keyword, isp := range ispMappings {
		if strings.Contains(orgUpper, keyword) {
			return isp
		}
	}

	return "unknown"
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
