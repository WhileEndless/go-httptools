package rawhttp_test

import (
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/rawhttp"
)

func TestNewResponse(t *testing.T) {
	resp := rawhttp.NewResponse()

	if resp == nil {
		t.Fatal("NewResponse returned nil")
	}

	if resp.Headers == nil {
		t.Error("Headers not initialized")
	}

	if resp.Timing == nil {
		t.Error("Timing not initialized")
	}
}

func TestResponseHeaderMethods(t *testing.T) {
	resp := rawhttp.NewResponse()

	// Test SetHeader
	resp.SetHeader("Content-Type", "application/json")
	if got := resp.GetHeader("Content-Type"); got != "application/json" {
		t.Errorf("GetHeader = %q, want %q", got, "application/json")
	}

	// Test AddHeader
	resp.AddHeader("Set-Cookie", "session=abc123")
	resp.AddHeader("Set-Cookie", "token=xyz789")

	cookies := resp.GetHeaders("Set-Cookie")
	if len(cookies) != 2 {
		t.Errorf("GetHeaders length = %d, want 2", len(cookies))
	}

	if cookies[0] != "session=abc123" {
		t.Errorf("First cookie = %q, want %q", cookies[0], "session=abc123")
	}

	if cookies[1] != "token=xyz789" {
		t.Errorf("Second cookie = %q, want %q", cookies[1], "token=xyz789")
	}
}

func TestResponseStatusChecks(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		wantSuccessful bool
		wantRedirect   bool
		wantClientErr  bool
		wantServerErr  bool
	}{
		{
			name:           "200 OK",
			statusCode:     200,
			wantSuccessful: true,
		},
		{
			name:           "201 Created",
			statusCode:     201,
			wantSuccessful: true,
		},
		{
			name:         "301 Moved Permanently",
			statusCode:   301,
			wantRedirect: true,
		},
		{
			name:         "302 Found",
			statusCode:   302,
			wantRedirect: true,
		},
		{
			name:          "400 Bad Request",
			statusCode:    400,
			wantClientErr: true,
		},
		{
			name:          "404 Not Found",
			statusCode:    404,
			wantClientErr: true,
		},
		{
			name:          "500 Internal Server Error",
			statusCode:    500,
			wantServerErr: true,
		},
		{
			name:          "503 Service Unavailable",
			statusCode:    503,
			wantServerErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := rawhttp.NewResponse()
			resp.StatusCode = tt.statusCode

			if got := resp.IsSuccessful(); got != tt.wantSuccessful {
				t.Errorf("IsSuccessful() = %v, want %v", got, tt.wantSuccessful)
			}

			if got := resp.IsRedirect(); got != tt.wantRedirect {
				t.Errorf("IsRedirect() = %v, want %v", got, tt.wantRedirect)
			}

			if got := resp.IsClientError(); got != tt.wantClientErr {
				t.Errorf("IsClientError() = %v, want %v", got, tt.wantClientErr)
			}

			if got := resp.IsServerError(); got != tt.wantServerErr {
				t.Errorf("IsServerError() = %v, want %v", got, tt.wantServerErr)
			}
		})
	}
}

func TestResponseRawPreservation(t *testing.T) {
	rawData := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello, World!")

	resp := rawhttp.NewResponse()
	resp.Raw = rawData

	if len(resp.Raw) != len(rawData) {
		t.Errorf("Raw length = %d, want %d", len(resp.Raw), len(rawData))
	}

	for i, b := range resp.Raw {
		if b != rawData[i] {
			t.Errorf("Raw[%d] = %v, want %v", i, b, rawData[i])
		}
	}
}
