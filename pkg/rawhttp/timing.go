package rawhttp

import "time"

// Timing represents timing information for different phases of the request
type Timing struct {
	DNSLookup    time.Duration // Time spent on DNS resolution
	TCPConnect   time.Duration // Time spent on TCP connection establishment
	TLSHandshake time.Duration // Time spent on TLS handshake (0 for HTTP)
	TTFB         time.Duration // Time to first byte (from sending request to receiving first response byte)
	Total        time.Duration // Total time from start to finish
	ProxyConnect time.Duration // Time spent connecting to proxy (0 if no proxy)
}

// String returns a human-readable representation of timing information
func (t *Timing) String() string {
	return formatTiming(t)
}

func formatTiming(t *Timing) string {
	result := "Timing:\n"
	if t.DNSLookup > 0 {
		result += "  DNS Lookup: " + t.DNSLookup.String() + "\n"
	}
	if t.ProxyConnect > 0 {
		result += "  Proxy Connect: " + t.ProxyConnect.String() + "\n"
	}
	if t.TCPConnect > 0 {
		result += "  TCP Connect: " + t.TCPConnect.String() + "\n"
	}
	if t.TLSHandshake > 0 {
		result += "  TLS Handshake: " + t.TLSHandshake.String() + "\n"
	}
	if t.TTFB > 0 {
		result += "  Time to First Byte: " + t.TTFB.String() + "\n"
	}
	result += "  Total: " + t.Total.String()
	return result
}
