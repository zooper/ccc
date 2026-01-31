package monitor

import (
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// PingResult contains the result of a ping attempt
type PingResult struct {
	Success  bool
	RTT      time.Duration
	Error    error
}

// Pinger handles ICMP ping operations
type Pinger struct {
	timeout    time.Duration
	count      int
	privileged bool
}

// NewPinger creates a new Pinger
func NewPinger(timeout time.Duration, privileged bool) *Pinger {
	return &Pinger{
		timeout:    timeout,
		count:      3, // Send 3 pings
		privileged: privileged,
	}
}

// Ping sends ICMP echo requests to the specified IP
func (p *Pinger) Ping(ip string) PingResult {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		return PingResult{Success: false, Error: err}
	}

	pinger.Count = p.count
	pinger.Timeout = p.timeout
	pinger.SetPrivileged(p.privileged)

	err = pinger.Run()
	if err != nil {
		return PingResult{Success: false, Error: err}
	}

	stats := pinger.Statistics()

	// Consider successful if at least one packet was received
	success := stats.PacketsRecv > 0

	return PingResult{
		Success: success,
		RTT:     stats.AvgRtt,
		Error:   nil,
	}
}
