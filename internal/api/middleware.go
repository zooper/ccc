package api

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	TrustedProxies []string // IPs/CIDRs trusted to set X-Forwarded-For
	CORSOrigin     string   // Allowed CORS origin (empty = same-origin only)
	MaxBodySize    int64    // Maximum request body size in bytes
}

// DefaultSecurityConfig returns safe defaults
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		TrustedProxies: nil,           // Don't trust any proxy by default
		CORSOrigin:     "",            // Same-origin only
		MaxBodySize:    1024 * 1024,   // 1MB
	}
}

// parsedProxies caches parsed CIDR networks
type parsedProxies struct {
	ips   map[string]bool
	cidrs []*net.IPNet
}

var (
	proxyCache   *parsedProxies
	proxyCacheMu sync.RWMutex
)

func parseProxies(proxies []string) *parsedProxies {
	p := &parsedProxies{
		ips: make(map[string]bool),
	}
	for _, proxy := range proxies {
		proxy = strings.TrimSpace(proxy)
		if strings.Contains(proxy, "/") {
			// CIDR notation
			_, network, err := net.ParseCIDR(proxy)
			if err == nil {
				p.cidrs = append(p.cidrs, network)
			}
		} else {
			// Single IP
			if ip := net.ParseIP(proxy); ip != nil {
				p.ips[ip.String()] = true
			}
		}
	}
	return p
}

func isTrustedProxy(remoteIP string, proxies *parsedProxies) bool {
	if proxies == nil {
		return false
	}

	ip := net.ParseIP(remoteIP)
	if ip == nil {
		return false
	}

	// Check exact IP match
	if proxies.ips[ip.String()] {
		return true
	}

	// Check CIDR ranges
	for _, cidr := range proxies.cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Printf("%s %s %d %s [%s]",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			time.Since(start),
			GetClientIP(r),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// CORSMiddleware adds CORS headers based on configuration
func CORSMiddleware(cfg SecurityConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Only add CORS headers if there's an Origin header
			if origin != "" {
				allowedOrigin := ""

				if cfg.CORSOrigin == "*" {
					// Allow all origins (not recommended for production)
					allowedOrigin = "*"
				} else if cfg.CORSOrigin != "" {
					// Check if origin matches configured origin
					if origin == cfg.CORSOrigin {
						allowedOrigin = origin
					}
				}
				// If CORSOrigin is empty, don't set any CORS headers (same-origin only)

				if allowedOrigin != "" {
					w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
					w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

					if allowedOrigin != "*" {
						w.Header().Set("Vary", "Origin")
					}
				}
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// BodyLimitMiddleware limits request body size
func BodyLimitMiddleware(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxSize {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	tokens   map[string]*bucket
	rate     float64       // tokens per second
	burst    int           // max tokens
	cleanup  time.Duration // cleanup interval
	lastClean time.Time
}

type bucket struct {
	tokens    float64
	lastCheck time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(ratePerSecond float64, burst int) *RateLimiter {
	return &RateLimiter{
		tokens:   make(map[string]*bucket),
		rate:     ratePerSecond,
		burst:    burst,
		cleanup:  5 * time.Minute,
		lastClean: time.Now(),
	}
}

// Allow checks if a request from the given key should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Periodic cleanup of old buckets
	now := time.Now()
	if now.Sub(rl.lastClean) > rl.cleanup {
		for k, b := range rl.tokens {
			if now.Sub(b.lastCheck) > rl.cleanup {
				delete(rl.tokens, k)
			}
		}
		rl.lastClean = now
	}

	b, exists := rl.tokens[key]
	if !exists {
		b = &bucket{
			tokens:    float64(rl.burst),
			lastCheck: now,
		}
		rl.tokens[key] = b
	}

	// Add tokens based on time elapsed
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	b.lastCheck = now

	// Check if we have a token
	if b.tokens >= 1 {
		b.tokens--
		return true
	}

	return false
}

// RateLimitMiddleware applies rate limiting per client IP
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := GetClientIP(r)

			if !limiter.Allow(clientIP) {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SetTrustedProxies configures trusted proxy IPs/CIDRs
func SetTrustedProxies(proxies []string) {
	proxyCacheMu.Lock()
	defer proxyCacheMu.Unlock()
	proxyCache = parseProxies(proxies)
	if len(proxies) > 0 {
		log.Printf("Configured trusted proxies: %v", proxies)
	}
}

// GetClientIP extracts the client IP from the request
// Only trusts X-Forwarded-For/X-Real-IP from configured trusted proxies
func GetClientIP(r *http.Request) string {
	// Get the direct connection IP
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	// Check if remote IP is a trusted proxy
	proxyCacheMu.RLock()
	trusted := isTrustedProxy(remoteIP, proxyCache)
	proxyCacheMu.RUnlock()

	if !trusted {
		// Not from a trusted proxy, use direct connection IP
		return remoteIP
	}

	// Check X-Forwarded-For header (might contain multiple IPs)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (original client)
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if isValidIP(ip) {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if isValidIP(xri) {
			return xri
		}
	}

	// Fall back to remote IP
	return remoteIP
}

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}
