package rawhttp_test

import (
	"strings"
	"testing"
	"time"

	"github.com/WhileEndless/go-httptools/pkg/rawhttp"
)

func TestTimingString(t *testing.T) {
	timing := &rawhttp.Timing{
		DNSLookup:    10 * time.Millisecond,
		TCPConnect:   20 * time.Millisecond,
		TLSHandshake: 50 * time.Millisecond,
		TTFB:         100 * time.Millisecond,
		Total:        200 * time.Millisecond,
	}

	result := timing.String()

	if result == "" {
		t.Error("String() returned empty string")
	}

	// Should contain all timing components
	expectedParts := []string{"DNS Lookup", "TCP Connect", "TLS Handshake", "Time to First Byte", "Total"}
	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("String() missing %q", part)
		}
	}
}

func TestTimingStringWithProxy(t *testing.T) {
	timing := &rawhttp.Timing{
		ProxyConnect: 30 * time.Millisecond,
		TCPConnect:   20 * time.Millisecond,
		Total:        100 * time.Millisecond,
	}

	result := timing.String()

	if !strings.Contains(result, "Proxy Connect") {
		t.Error("String() missing proxy connect timing")
	}
}

func TestTimingStringHTTPOnly(t *testing.T) {
	timing := &rawhttp.Timing{
		DNSLookup:  10 * time.Millisecond,
		TCPConnect: 20 * time.Millisecond,
		TTFB:       100 * time.Millisecond,
		Total:      150 * time.Millisecond,
		// No TLS handshake for HTTP
	}

	result := timing.String()

	// Should NOT contain TLS handshake for HTTP-only request
	if strings.Contains(result, "TLS Handshake: 0") {
		t.Error("String() should not show TLS Handshake with 0 duration")
	}
}
