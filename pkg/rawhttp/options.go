package rawhttp

import (
	"crypto/tls"
	"crypto/x509"
	"time"
)

// Options represents configuration options for sending HTTP requests
type Options struct {
	// Connection options
	Scheme string // "http" or "https"
	Host   string // Target hostname
	Port   int    // Target port (default: 80 for HTTP, 443 for HTTPS)
	ConnIP string // Specific IP to connect to (bypasses DNS if set)

	// Timeout options
	ConnTimeout  time.Duration // Connection timeout (default: 30s)
	ReadTimeout  time.Duration // Read timeout (default: 30s)
	WriteTimeout time.Duration // Write timeout (default: 30s)

	// TLS options
	DisableSNI         bool     // Disable SNI (Server Name Indication)
	InsecureSkipVerify bool     // Skip TLS certificate verification
	CustomCACerts      [][]byte // Custom CA certificates in PEM format

	// Body options
	BodyMemLimit int64 // Maximum body size to keep in memory (default: 4MB)

	// Connection pooling
	ReuseConnection bool // Enable connection pooling/Keep-Alive (default: true)

	// Proxy options
	ProxyURL string // Upstream proxy URL (e.g., "http://proxy:8080" or "socks5://proxy:1080")

	// Protocol options
	ForceHTTP1 bool // Force HTTP/1.1 even if HTTP/2 is available
	ForceHTTP2 bool // Force HTTP/2 (will fail if not supported)
	EnableH2C  bool // Enable HTTP/2 cleartext (H2C) for non-TLS connections
}

// SetDefaults sets default values for unspecified options
func (o *Options) SetDefaults() {
	if o.Port == 0 {
		if o.Scheme == "https" {
			o.Port = 443
		} else {
			o.Port = 80
		}
	}

	if o.ConnTimeout == 0 {
		o.ConnTimeout = 30 * time.Second
	}

	if o.ReadTimeout == 0 {
		o.ReadTimeout = 30 * time.Second
	}

	if o.WriteTimeout == 0 {
		o.WriteTimeout = 30 * time.Second
	}

	if o.BodyMemLimit == 0 {
		o.BodyMemLimit = 4 * 1024 * 1024 // 4MB
	}

	// Connection reuse is enabled by default
	if !o.ForceHTTP1 && !o.ForceHTTP2 {
		o.ReuseConnection = true
	}
}

// BuildTLSConfig builds a TLS configuration from options
func (o *Options) BuildTLSConfig() *tls.Config {
	config := &tls.Config{
		InsecureSkipVerify: o.InsecureSkipVerify,
	}

	// Set SNI
	if !o.DisableSNI {
		config.ServerName = o.Host
	}

	// Add custom CA certificates
	if len(o.CustomCACerts) > 0 {
		certPool := x509.NewCertPool()
		for _, cert := range o.CustomCACerts {
			certPool.AppendCertsFromPEM(cert)
		}
		config.RootCAs = certPool
	}

	// Configure ALPN for HTTP/2
	if !o.ForceHTTP1 {
		config.NextProtos = []string{"h2", "http/1.1"}
	} else {
		config.NextProtos = []string{"http/1.1"}
	}

	if o.ForceHTTP2 {
		config.NextProtos = []string{"h2"}
	}

	return config
}

