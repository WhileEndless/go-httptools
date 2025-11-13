package rawhttp_test

import (
	"testing"
	"time"

	"github.com/WhileEndless/go-httptools/pkg/rawhttp"
)

func TestOptionsSetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		opts     rawhttp.Options
		wantPort int
	}{
		{
			name:     "HTTP default port",
			opts:     rawhttp.Options{Scheme: "http"},
			wantPort: 80,
		},
		{
			name:     "HTTPS default port",
			opts:     rawhttp.Options{Scheme: "https"},
			wantPort: 443,
		},
		{
			name:     "Custom port preserved",
			opts:     rawhttp.Options{Scheme: "http", Port: 8080},
			wantPort: 8080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.opts.SetDefaults()

			if tt.opts.Port != tt.wantPort {
				t.Errorf("Port = %d, want %d", tt.opts.Port, tt.wantPort)
			}

			if tt.opts.ConnTimeout == 0 {
				t.Error("ConnTimeout not set")
			}

			if tt.opts.ReadTimeout == 0 {
				t.Error("ReadTimeout not set")
			}

			if tt.opts.WriteTimeout == 0 {
				t.Error("WriteTimeout not set")
			}

			if tt.opts.BodyMemLimit == 0 {
				t.Error("BodyMemLimit not set")
			}
		})
	}
}

func TestBuildTLSConfig(t *testing.T) {
	tests := []struct {
		name               string
		opts               rawhttp.Options
		wantInsecureSkip   bool
		wantServerName     string
		wantNextProtos     []string
	}{
		{
			name: "Default TLS config",
			opts: rawhttp.Options{
				Host: "example.com",
			},
			wantInsecureSkip: false,
			wantServerName:   "example.com",
			wantNextProtos:   []string{"h2", "http/1.1"},
		},
		{
			name: "Insecure skip verify",
			opts: rawhttp.Options{
				Host:               "example.com",
				InsecureSkipVerify: true,
			},
			wantInsecureSkip: true,
			wantServerName:   "example.com",
			wantNextProtos:   []string{"h2", "http/1.1"},
		},
		{
			name: "SNI disabled",
			opts: rawhttp.Options{
				Host:       "example.com",
				DisableSNI: true,
			},
			wantInsecureSkip: false,
			wantServerName:   "",
			wantNextProtos:   []string{"h2", "http/1.1"},
		},
		{
			name: "Force HTTP/1.1",
			opts: rawhttp.Options{
				Host:        "example.com",
				ForceHTTP1:  true,
			},
			wantInsecureSkip: false,
			wantServerName:   "example.com",
			wantNextProtos:   []string{"http/1.1"},
		},
		{
			name: "Force HTTP/2",
			opts: rawhttp.Options{
				Host:        "example.com",
				ForceHTTP2:  true,
			},
			wantInsecureSkip: false,
			wantServerName:   "example.com",
			wantNextProtos:   []string{"h2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.opts.BuildTLSConfig()

			if config.InsecureSkipVerify != tt.wantInsecureSkip {
				t.Errorf("InsecureSkipVerify = %v, want %v", config.InsecureSkipVerify, tt.wantInsecureSkip)
			}

			if config.ServerName != tt.wantServerName {
				t.Errorf("ServerName = %q, want %q", config.ServerName, tt.wantServerName)
			}

			if len(config.NextProtos) != len(tt.wantNextProtos) {
				t.Errorf("NextProtos length = %d, want %d", len(config.NextProtos), len(tt.wantNextProtos))
			}

			for i, proto := range config.NextProtos {
				if proto != tt.wantNextProtos[i] {
					t.Errorf("NextProtos[%d] = %q, want %q", i, proto, tt.wantNextProtos[i])
				}
			}
		})
	}
}

func TestOptionsTimeouts(t *testing.T) {
	opts := rawhttp.Options{
		Scheme:       "https",
		Host:         "example.com",
		ConnTimeout:  10 * time.Second,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	opts.SetDefaults()

	if opts.ConnTimeout != 10*time.Second {
		t.Errorf("ConnTimeout = %v, want %v", opts.ConnTimeout, 10*time.Second)
	}

	if opts.ReadTimeout != 20*time.Second {
		t.Errorf("ReadTimeout = %v, want %v", opts.ReadTimeout, 20*time.Second)
	}

	if opts.WriteTimeout != 5*time.Second {
		t.Errorf("WriteTimeout = %v, want %v", opts.WriteTimeout, 5*time.Second)
	}
}
