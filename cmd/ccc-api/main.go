package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jonsson/ccc/internal/api"
	"github.com/jonsson/ccc/internal/isp"
	"github.com/jonsson/ccc/internal/monitor"
	"github.com/jonsson/ccc/internal/storage"
)

//go:embed static
var staticFiles embed.FS

// Config holds application configuration
type Config struct {
	DBPath        string
	ListenAddr    string
	PingInterval  time.Duration
	ExpireDays    int
	Privileged    bool
	SetPassword   string   // If set, just set the password and exit
	TrustedProxies []string // IPs/CIDRs trusted to set X-Forwarded-For
	CORSOrigin    string   // Allowed CORS origin (empty = same-origin only)
	ISPConfigPath string   // Path to ISP config JSON file
}

func main() {
	cfg := parseConfig()

	// Initialize database (needed for both server and password setting)
	db, err := storage.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Handle set-password command
	if cfg.SetPassword != "" {
		if err := db.SetAdminPassword(cfg.SetPassword); err != nil {
			log.Fatalf("Failed to set admin password: %v", err)
		}
		fmt.Println("Admin password set successfully.")
		return
	}

	// Check if admin password is configured
	hasPassword, err := db.HasAdminPassword()
	if err != nil {
		log.Fatalf("Failed to check admin password: %v", err)
	}
	if !hasPassword {
		log.Println("WARNING: No admin password set. Run with --set-password <password> to set one.")
	}

	log.Printf("CCC API Server v%s", api.Version)
	log.Printf("Database: %s", cfg.DBPath)
	log.Printf("Listen address: %s", cfg.ListenAddr)

	// Initialize ISP classifier
	classifier := isp.NewClassifier()
	if cfg.ISPConfigPath != "" {
		if err := classifier.LoadConfig(cfg.ISPConfigPath); err != nil {
			log.Fatalf("Failed to load ISP config: %v", err)
		}
	} else {
		log.Println("WARNING: No ISP config file specified. Using fallback ASN org names.")
	}

	// Initialize pinger
	pinger := monitor.NewPinger(5*time.Second, cfg.Privileged)

	// Initialize scheduler
	scheduler := monitor.NewScheduler(db, pinger, cfg.PingInterval, cfg.ExpireDays)

	// Setup HTTP server
	handler := api.NewHandler(db, cfg.DBPath, classifier)
	handler.SetMetricsProvider(scheduler) // Connect handler with scheduler for metrics
	mux := http.NewServeMux()

	// Try to get embedded static files
	var staticFS fs.FS
	subFS, err := fs.Sub(staticFiles, "static")
	if err == nil {
		// Check if index.html exists
		if _, err := subFS.Open("index.html"); err == nil {
			staticFS = subFS
			log.Println("Serving embedded static files")
		}
	}

	handler.SetupRoutes(mux, staticFS)

	// Configure security settings
	api.SetTrustedProxies(cfg.TrustedProxies)

	securityCfg := api.SecurityConfig{
		TrustedProxies: cfg.TrustedProxies,
		CORSOrigin:     cfg.CORSOrigin,
		MaxBodySize:    1024 * 1024, // 1MB
	}

	// Create rate limiters
	// General API: 100 requests per second, burst of 200
	generalLimiter := api.NewRateLimiter(100, 200)
	// Auth endpoints: 5 requests per second, burst of 10 (prevent brute force)
	authLimiter := api.NewRateLimiter(5, 10)

	// Apply middleware (order matters: outermost first)
	var httpHandler http.Handler = mux
	httpHandler = api.RateLimitMiddleware(generalLimiter)(httpHandler)
	httpHandler = api.BodyLimitMiddleware(securityCfg.MaxBodySize)(httpHandler)
	httpHandler = api.CORSMiddleware(securityCfg)(httpHandler)
	httpHandler = api.LoggingMiddleware(httpHandler)

	// Set auth rate limiter on handler for admin endpoints
	handler.SetAuthRateLimiter(authLimiter)

	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      httpHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start monitoring in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)

	// Handle shutdown gracefully
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down...")
		cancel()
		scheduler.Stop()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	// Start HTTP server
	log.Printf("Starting HTTP server on %s", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}

	log.Println("Server stopped")
}

func parseConfig() Config {
	cfg := Config{}

	var trustedProxies string

	flag.StringVar(&cfg.DBPath, "db", getEnv("CCC_DB_PATH", "./ccc.db"), "Database file path")
	flag.StringVar(&cfg.ListenAddr, "listen", getEnv("CCC_LISTEN_ADDR", ":8080"), "Listen address")
	flag.DurationVar(&cfg.PingInterval, "ping-interval", getEnvDuration("CCC_PING_INTERVAL", 60*time.Second), "Ping interval")
	flag.IntVar(&cfg.ExpireDays, "expire-days", getEnvInt("CCC_EXPIRE_DAYS", 30), "Days before endpoint expiry")
	flag.BoolVar(&cfg.Privileged, "privileged", getEnvBool("CCC_PRIVILEGED", false), "Use privileged (raw socket) ICMP")
	flag.StringVar(&cfg.SetPassword, "set-password", "", "Set admin password and exit")
	flag.StringVar(&trustedProxies, "trusted-proxies", getEnv("CCC_TRUSTED_PROXIES", ""), "Comma-separated list of trusted proxy IPs/CIDRs (e.g., 127.0.0.1,::1,10.0.0.0/8)")
	flag.StringVar(&cfg.CORSOrigin, "cors-origin", getEnv("CCC_CORS_ORIGIN", ""), "Allowed CORS origin (empty = same-origin only)")
	flag.StringVar(&cfg.ISPConfigPath, "isp-config", getEnv("CCC_ISP_CONFIG", ""), "Path to ISP config JSON file")

	flag.Parse()

	// Parse trusted proxies
	if trustedProxies != "" {
		for _, p := range strings.Split(trustedProxies, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.TrustedProxies = append(cfg.TrustedProxies, p)
			}
		}
	}

	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		result := 0
		for _, c := range val {
			if c < '0' || c > '9' {
				return defaultVal
			}
			result = result*10 + int(c-'0')
		}
		return result
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultVal
}
