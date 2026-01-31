package monitor

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// Hop represents a single hop in a traceroute
type Hop struct {
	TTL     int
	Address string
	RTT     time.Duration
	Reached bool // True if this is the final destination
}

// TracerouteResult contains the result of a traceroute
type TracerouteResult struct {
	Hops       []Hop
	LastHop    *Hop   // The last hop that responded
	ReachedDst bool   // True if we reached the destination
	Error      error
}

// Tracer handles traceroute operations
type Tracer struct {
	timeout    time.Duration
	maxHops    int
	probes     int // Number of probes per hop
}

// NewTracer creates a new Tracer
func NewTracer(timeout time.Duration, maxHops int) *Tracer {
	return &Tracer{
		timeout: timeout,
		maxHops: maxHops,
		probes:  1, // Single probe per hop for efficiency
	}
}

// Traceroute performs a traceroute to the specified IP address
func (t *Tracer) Traceroute(destIP string) TracerouteResult {
	dst := net.ParseIP(destIP)
	if dst == nil {
		return TracerouteResult{Error: fmt.Errorf("invalid IP address: %s", destIP)}
	}

	// Use ICMP (requires root/CAP_NET_RAW)
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return TracerouteResult{Error: fmt.Errorf("failed to listen: %w", err)}
	}
	defer conn.Close()

	var hops []Hop
	var lastResponding *Hop

	for ttl := 1; ttl <= t.maxHops; ttl++ {
		hop := t.probeHop(conn, dst, ttl)
		hops = append(hops, hop)

		if hop.Address != "" {
			hopCopy := hop
			lastResponding = &hopCopy
		}

		if hop.Reached {
			// We've reached the destination
			return TracerouteResult{
				Hops:       hops,
				LastHop:    lastResponding,
				ReachedDst: true,
			}
		}
	}

	return TracerouteResult{
		Hops:       hops,
		LastHop:    lastResponding,
		ReachedDst: false,
	}
}

func (t *Tracer) probeHop(conn *icmp.PacketConn, dst net.IP, ttl int) Hop {
	hop := Hop{TTL: ttl}

	// Set TTL
	if err := conn.IPv4PacketConn().SetTTL(ttl); err != nil {
		return hop
	}

	// Create ICMP echo request
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   ttl, // Use TTL as ID for simplicity
			Seq:  1,
			Data: []byte("CCC-TRACE"),
		},
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return hop
	}

	start := time.Now()

	// Send the packet
	if _, err := conn.WriteTo(msgBytes, &net.IPAddr{IP: dst}); err != nil {
		return hop
	}

	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(t.timeout)); err != nil {
		return hop
	}

	// Read response
	reply := make([]byte, 1500)
	n, peer, err := conn.ReadFrom(reply)
	if err != nil {
		// Timeout or other error - no response at this hop
		return hop
	}

	hop.RTT = time.Since(start)
	hop.Address = peer.String()

	// Parse the ICMP response
	rm, err := icmp.ParseMessage(1, reply[:n]) // 1 = ICMP for IPv4
	if err != nil {
		return hop
	}

	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		// We've reached the destination
		hop.Reached = true
	case ipv4.ICMPTypeTimeExceeded:
		// Intermediate hop (TTL expired in transit)
		hop.Reached = false
	case ipv4.ICMPTypeDestinationUnreachable:
		// Destination unreachable but we know there's a hop here
		hop.Reached = true // Consider this as reaching the edge
	}

	return hop
}

// FindLastRespondingHop returns the IP of the last hop that responded
// This is useful when the destination doesn't respond to ICMP
func (t *Tracer) FindLastRespondingHop(destIP string) (hopIP string, hopNum int, reached bool) {
	result := t.Traceroute(destIP)
	if result.Error != nil {
		return "", 0, false
	}

	if result.LastHop == nil {
		return "", 0, false
	}

	return result.LastHop.Address, result.LastHop.TTL, result.ReachedDst
}
