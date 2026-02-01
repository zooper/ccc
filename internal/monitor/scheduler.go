package monitor

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jonsson/ccc/internal/models"
	"github.com/jonsson/ccc/internal/storage"
)

// Scheduler manages periodic monitoring tasks
type Scheduler struct {
	db           *storage.DB
	pinger       *Pinger
	pingInterval time.Duration
	expireDays   int
	stopCh       chan struct{}
	wg           sync.WaitGroup

	// Outage analysis results (updated after each ping cycle)
	outagesMu sync.RWMutex
	outages   map[string]bool // ISP -> likely outage

	// Last ping cycle timestamp
	lastPingMu   sync.RWMutex
	lastPingTime time.Time

	// Metrics
	startTime      time.Time
	pingCycleCount int64
	pingCycleMu    sync.RWMutex
}

// NewScheduler creates a new monitoring scheduler
func NewScheduler(db *storage.DB, pinger *Pinger, pingInterval time.Duration, expireDays int) *Scheduler {
	return &Scheduler{
		db:           db,
		pinger:       pinger,
		pingInterval: pingInterval,
		expireDays:   expireDays,
		stopCh:       make(chan struct{}),
		startTime:    time.Now(),
	}
}

// Start begins the monitoring loops
func (s *Scheduler) Start(ctx context.Context) {
	log.Printf("Starting monitoring scheduler (interval: %s, expire: %d days)",
		s.pingInterval, s.expireDays)

	// Start ping loop
	s.wg.Add(1)
	go s.pingLoop(ctx)

	// Start cleanup loop (runs daily)
	s.wg.Add(1)
	go s.cleanupLoop(ctx)

	// Run initial cleanup
	s.runCleanup()
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	log.Println("Monitoring scheduler stopped")
}

func (s *Scheduler) pingLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.pingInterval)
	defer ticker.Stop()

	// Run immediately on start
	s.runPingCycle()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.runPingCycle()
		}
	}
}

// pingResult holds the result of pinging an endpoint
type pingResult struct {
	endpoint  models.Endpoint
	oldStatus string
	newStatus string
	lastOK    time.Time
}

func (s *Scheduler) runPingCycle() {
	endpoints, err := s.db.ListAll()
	if err != nil {
		log.Printf("Failed to list endpoints for ping cycle: %v", err)
		return
	}

	if len(endpoints) == 0 {
		return
	}

	log.Printf("Starting ping cycle for %d endpoints (parallel)", len(endpoints))

	// Use worker pool for parallel pinging
	numWorkers := 50 // Concurrent ping workers
	if numWorkers > len(endpoints) {
		numWorkers = len(endpoints)
	}

	jobs := make(chan models.Endpoint, len(endpoints))
	results := make(chan pingResult, len(endpoints))

	// Start workers
	var workerWg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			for ep := range jobs {
				oldStatus := ep.Status
				status, lastOK := s.monitorEndpoint(&ep)
				results <- pingResult{
					endpoint:  ep,
					oldStatus: oldStatus,
					newStatus: status,
					lastOK:    lastOK,
				}
			}
		}()
	}

	// Send jobs
	for _, ep := range endpoints {
		jobs <- ep
	}
	close(jobs)

	// Wait for workers to finish, then close results
	go func() {
		workerWg.Wait()
		close(results)
	}()

	// Collect results
	var upCount, downCount int
	for result := range results {
		if result.newStatus == "up" {
			upCount++
		} else {
			downCount++
		}

		// Record status change events
		if result.oldStatus != result.newStatus && result.oldStatus != "unknown" {
			if result.newStatus == "down" {
				msg := result.endpoint.ISP + " endpoint went down"
				if err := s.db.RecordEvent("down", result.endpoint.ISP, result.endpoint.ID, msg); err != nil {
					log.Printf("Failed to record down event: %v", err)
				}
			} else if result.newStatus == "up" && result.oldStatus == "down" {
				msg := result.endpoint.ISP + " endpoint recovered"
				if err := s.db.RecordEvent("up", result.endpoint.ISP, result.endpoint.ID, msg); err != nil {
					log.Printf("Failed to record up event: %v", err)
				}
			}
		}

		if err := s.db.UpdateStatus(result.endpoint.ID, result.newStatus, result.lastOK); err != nil {
			log.Printf("Failed to update status for %s: %v", result.endpoint.ID, err)
		}
	}

	log.Printf("Ping cycle complete: %d up, %d down", upCount, downCount)

	// Record ping cycle completion time and increment counter
	s.lastPingMu.Lock()
	s.lastPingTime = time.Now()
	s.lastPingMu.Unlock()

	s.pingCycleMu.Lock()
	s.pingCycleCount++
	s.pingCycleMu.Unlock()

	// Record uptime history
	if err := s.db.RecordUptimeSnapshot(len(endpoints), upCount, downCount); err != nil {
		log.Printf("Failed to record uptime snapshot: %v", err)
	}

	// Cleanup old history (keep 7 days)
	if deleted, err := s.db.CleanupOldHistory(7 * 24 * time.Hour); err != nil {
		log.Printf("Failed to cleanup old history: %v", err)
	} else if deleted > 0 {
		log.Printf("Cleaned up %d old history records", deleted)
	}

	// Cleanup old events (keep 7 days)
	if deleted, err := s.db.CleanupOldEvents(7 * 24 * time.Hour); err != nil {
		log.Printf("Failed to cleanup old events: %v", err)
	} else if deleted > 0 {
		log.Printf("Cleaned up %d old events", deleted)
	}

	// Analyze for ISP-level outages
	oldOutages := s.outages
	outages := s.analyzeISPOutages()

	// Record outage/recovery events
	for isp, isOutage := range outages {
		wasOutage := oldOutages[isp]
		if isOutage && !wasOutage {
			msg := isp + " ISP outage detected"
			if err := s.db.RecordEvent("outage", isp, "", msg); err != nil {
				log.Printf("Failed to record outage event: %v", err)
			}
		} else if !isOutage && wasOutage {
			msg := isp + " ISP recovered from outage"
			if err := s.db.RecordEvent("recovery", isp, "", msg); err != nil {
				log.Printf("Failed to record recovery event: %v", err)
			}
		}
	}

	s.outagesMu.Lock()
	s.outages = outages
	s.outagesMu.Unlock()
}

// LastPingTime returns the time of the last completed ping cycle
func (s *Scheduler) LastPingTime() time.Time {
	s.lastPingMu.RLock()
	defer s.lastPingMu.RUnlock()
	return s.lastPingTime
}

// PingInterval returns the configured ping interval
func (s *Scheduler) PingInterval() time.Duration {
	return s.pingInterval
}

// NextPingTime returns the estimated time of the next ping cycle
func (s *Scheduler) NextPingTime() time.Time {
	s.lastPingMu.RLock()
	defer s.lastPingMu.RUnlock()
	if s.lastPingTime.IsZero() {
		return time.Now()
	}
	return s.lastPingTime.Add(s.pingInterval)
}

// PingCycleCount returns the total number of ping cycles completed
func (s *Scheduler) PingCycleCount() int64 {
	s.pingCycleMu.RLock()
	defer s.pingCycleMu.RUnlock()
	return s.pingCycleCount
}

// StartTime returns when the scheduler started
func (s *Scheduler) StartTime() time.Time {
	return s.startTime
}

// IsISPOutage returns true if the specified ISP is likely experiencing an outage
func (s *Scheduler) IsISPOutage(isp string) bool {
	s.outagesMu.RLock()
	defer s.outagesMu.RUnlock()
	return s.outages[isp]
}

// HasAnyOutage returns true if any ISP is likely experiencing an outage
func (s *Scheduler) HasAnyOutage() bool {
	s.outagesMu.RLock()
	defer s.outagesMu.RUnlock()
	for _, outage := range s.outages {
		if outage {
			return true
		}
	}
	return false
}

// monitorEndpoint monitors a single endpoint via direct ping
func (s *Scheduler) monitorEndpoint(ep *models.Endpoint) (status string, lastOK time.Time) {
	result := s.pinger.Ping(ep.IPv4)

	if result.Success {
		return "up", time.Now()
	}

	// Ping failed - mark as unreachable (user can still view dashboard)
	if result.Error != nil {
		log.Printf("Ping failed for %s (%s): %v", ep.ID, ep.ISP, result.Error)
	}

	return "unreachable", time.Time{}
}

// analyzeISPOutages checks for common hop failures across endpoints from the same ISP
// Returns a map of ISP -> likely outage (true if multiple endpoints share a failing hop)
func (s *Scheduler) analyzeISPOutages() map[string]bool {
	endpoints, err := s.db.ListAll()
	if err != nil {
		log.Printf("Failed to analyze ISP outages: %v", err)
		return nil
	}

	// Group endpoints by ISP
	byISP := make(map[string][]models.Endpoint)
	for _, ep := range endpoints {
		byISP[ep.ISP] = append(byISP[ep.ISP], ep)
	}

	outages := make(map[string]bool)

	for isp, eps := range byISP {
		if len(eps) < 2 {
			// Need at least 2 endpoints to compare
			continue
		}

		// Count how many are down and using hops
		downCount := 0
		hopDownCount := 0
		sharedHops := make(map[string]int) // hop IP -> count of endpoints using it

		for _, ep := range eps {
			if ep.Status == "down" {
				downCount++
				if ep.UseHop && ep.MonitoredHop != "" {
					hopDownCount++
					sharedHops[ep.MonitoredHop]++
				}
			}
			// Also track all shared hops (even for up endpoints)
			if ep.UseHop && ep.MonitoredHop != "" {
				sharedHops[ep.MonitoredHop]++
			}
		}

		// Heuristic: If >50% of endpoints are down, likely ISP outage
		if float64(downCount)/float64(len(eps)) > 0.5 {
			outages[isp] = true
			log.Printf("Likely %s outage: %d/%d endpoints down", isp, downCount, len(eps))
			continue
		}

		// Check if multiple endpoints share the same failing hop
		for hop, count := range sharedHops {
			if count >= 2 {
				// Check if this shared hop is down for all users
				hopEndpoints, _ := s.db.GetEndpointsByMonitoredHop(hop)
				allDown := true
				for _, he := range hopEndpoints {
					if he.Status == "up" {
						allDown = false
						break
					}
				}
				if allDown && len(hopEndpoints) >= 2 {
					outages[isp] = true
					log.Printf("Likely %s outage: shared hop %s down for %d endpoints", isp, hop, count)
				}
			}
		}
	}

	return outages
}

func (s *Scheduler) cleanupLoop(ctx context.Context) {
	defer s.wg.Done()

	// Run daily
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.runCleanup()
		}
	}
}

func (s *Scheduler) runCleanup() {
	deleted, err := s.db.DeleteExpired(s.expireDays)
	if err != nil {
		log.Printf("Failed to cleanup expired endpoints: %v", err)
		return
	}

	if deleted > 0 {
		log.Printf("Cleaned up %d expired endpoints (not seen in %d days)", deleted, s.expireDays)
	}
}
