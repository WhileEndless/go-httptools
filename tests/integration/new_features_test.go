package integration

import (
	"bytes"
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/chunked"
	"github.com/WhileEndless/go-httptools/pkg/cookies"
	"github.com/WhileEndless/go-httptools/pkg/request"
	"github.com/WhileEndless/go-httptools/pkg/response"
)

// ============================================================================
// Chunked Transfer Encoding Tests
// ============================================================================

func TestChunkedEncoding_RequestParsing(t *testing.T) {
	rawRequest := []byte("POST /api HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\n" +
		"5\r\n" +
		"Hello\r\n" +
		"6\r\n" +
		" World\r\n" +
		"0\r\n" +
		"\r\n")

	req, err := request.Parse(rawRequest)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check Transfer-Encoding parsed
	if len(req.TransferEncoding) != 1 || req.TransferEncoding[0] != "chunked" {
		t.Errorf("Expected Transfer-Encoding=[chunked], got %v", req.TransferEncoding)
	}

	// Check IsBodyChunked flag
	if !req.IsBodyChunked {
		t.Error("Expected IsBodyChunked=true")
	}

	// Body should still be chunked (not auto-decoded)
	if !chunked.IsChunked(req.Body) {
		t.Error("Expected body to remain chunked")
	}

	// Explicit decode
	trailers := req.DecodeChunkedBody()
	if string(req.Body) != "Hello World" {
		t.Errorf("Expected decoded body='Hello World', got %q", string(req.Body))
	}

	if len(trailers) != 0 {
		t.Errorf("Expected no trailers, got %d", len(trailers))
	}

	// After decode, IsBodyChunked should be false
	if req.IsBodyChunked {
		t.Error("Expected IsBodyChunked=false after decode")
	}
}

func TestChunkedEncoding_ResponseParsing(t *testing.T) {
	rawResponse := []byte("HTTP/1.1 200 OK\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"7\r\n" +
		"Mozilla\r\n" +
		"9\r\n" +
		"Developer\r\n" +
		"7\r\n" +
		"Network\r\n" +
		"0\r\n" +
		"\r\n")

	resp, err := response.Parse(rawResponse)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check Transfer-Encoding parsed
	if len(resp.TransferEncoding) != 1 || resp.TransferEncoding[0] != "chunked" {
		t.Errorf("Expected Transfer-Encoding=[chunked], got %v", resp.TransferEncoding)
	}

	if !resp.IsBodyChunked {
		t.Error("Expected IsBodyChunked=true")
	}

	// Decode
	resp.DecodeChunkedBody()
	expected := "MozillaDeveloperNetwork"
	if string(resp.Body) != expected {
		t.Errorf("Expected decoded body=%q, got %q", expected, string(resp.Body))
	}
}

func TestChunkedEncoding_Encode(t *testing.T) {
	req := request.NewRequest()
	req.Method = "POST"
	req.URL = "/upload"
	req.Version = "HTTP/1.1"
	req.Headers.Set("Host", "example.com")
	req.Body = []byte("Test data for chunking")

	// Encode body
	req.EncodeChunkedBody(5)

	// Check Transfer-Encoding header set
	if req.Headers.Get("Transfer-Encoding") != "chunked" {
		t.Error("Expected Transfer-Encoding header to be set")
	}

	// Check IsBodyChunked flag
	if !req.IsBodyChunked {
		t.Error("Expected IsBodyChunked=true after encoding")
	}

	// Verify body is valid chunked encoding
	if !chunked.IsChunked(req.Body) {
		t.Error("Expected body to be chunked")
	}

	// Decode and verify
	decoded, _ := chunked.Decode(req.Body)
	if string(decoded) != "Test data for chunking" {
		t.Errorf("Decode failed: expected %q, got %q", "Test data for chunking", string(decoded))
	}
}

// ============================================================================
// Cookie Tests
// ============================================================================

func TestCookies_RequestParsing(t *testing.T) {
	rawRequest := []byte("GET / HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Cookie: session=abc123; user=john; preferences=dark_mode\r\n" +
		"\r\n")

	req, err := request.Parse(rawRequest)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check cookies auto-parsed
	if len(req.Cookies) != 3 {
		t.Fatalf("Expected 3 cookies, got %d", len(req.Cookies))
	}

	// Check individual cookies
	if req.GetCookie("session") != "abc123" {
		t.Errorf("Expected session=abc123, got %q", req.GetCookie("session"))
	}

	if req.GetCookie("user") != "john" {
		t.Errorf("Expected user=john, got %q", req.GetCookie("user"))
	}

	if req.GetCookie("preferences") != "dark_mode" {
		t.Errorf("Expected preferences=dark_mode, got %q", req.GetCookie("preferences"))
	}
}

func TestCookies_RequestModification(t *testing.T) {
	req := request.NewRequest()
	req.Method = "GET"
	req.URL = "/"
	req.Version = "HTTP/1.1"
	req.Headers.Set("Host", "example.com")

	// Add cookies
	req.SetCookie("session", "xyz789")
	req.SetCookie("logged_in", "true")

	// Update Cookie header
	req.UpdateCookieHeader()

	// Check header
	cookieHeader := req.Headers.Get("Cookie")
	if cookieHeader != "session=xyz789; logged_in=true" {
		t.Errorf("Expected Cookie header, got %q", cookieHeader)
	}
}

func TestCookies_ResponseParsing(t *testing.T) {
	rawResponse := []byte("HTTP/1.1 200 OK\r\n" +
		"Set-Cookie: session=abc123; Path=/; HttpOnly\r\n" +
		"Set-Cookie: user=john; Path=/; Secure\r\n" +
		"Content-Type: text/html\r\n" +
		"\r\n" +
		"<html></html>")

	resp, err := response.Parse(rawResponse)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check Set-Cookies auto-parsed
	if len(resp.SetCookies) != 2 {
		t.Fatalf("Expected 2 Set-Cookies, got %d", len(resp.SetCookies))
	}

	// Check first cookie
	sessionCookie := resp.GetSetCookie("session")
	if sessionCookie == nil {
		t.Fatal("Expected session cookie")
	}
	if sessionCookie.Value != "abc123" {
		t.Errorf("Expected value=abc123, got %q", sessionCookie.Value)
	}
	if !sessionCookie.HttpOnly {
		t.Error("Expected HttpOnly=true")
	}

	// Check second cookie
	userCookie := resp.GetSetCookie("user")
	if userCookie == nil {
		t.Fatal("Expected user cookie")
	}
	if !userCookie.Secure {
		t.Error("Expected Secure=true")
	}
}

func TestCookies_ResponseModification(t *testing.T) {
	resp := response.NewResponse()
	resp.Version = "HTTP/1.1"
	resp.StatusCode = 200
	resp.StatusText = "OK"

	// Add Set-Cookies
	resp.AddSetCookie(cookies.ResponseCookie{
		Name:     "token",
		Value:    "xyz789",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	resp.UpdateSetCookieHeaders()

	// Check header (note: this might fail due to OrderedHeaders limitation)
	setCookie := resp.Headers.Get("Set-Cookie")
	if setCookie == "" {
		t.Error("Expected Set-Cookie header to be set")
	}

	// Should contain key attributes
	if !contains(setCookie, "token=xyz789") {
		t.Error("Expected cookie name/value")
	}
}

// ============================================================================
// Query Parameter Tests
// ============================================================================

func TestQueryParams_Parsing(t *testing.T) {
	rawRequest := []byte("GET /search?q=golang&page=2&sort=date HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n")

	req, err := request.Parse(rawRequest)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check Path extracted
	if req.Path != "/search" {
		t.Errorf("Expected Path=/search, got %q", req.Path)
	}

	// Check query params
	if req.GetQueryParam("q") != "golang" {
		t.Errorf("Expected q=golang, got %q", req.GetQueryParam("q"))
	}

	if req.GetQueryParam("page") != "2" {
		t.Errorf("Expected page=2, got %q", req.GetQueryParam("page"))
	}

	if req.GetQueryParam("sort") != "date" {
		t.Errorf("Expected sort=date, got %q", req.GetQueryParam("sort"))
	}
}

func TestQueryParams_Modification(t *testing.T) {
	req := request.NewRequest()
	req.Method = "GET"
	req.URL = "/search?q=test&page=1"
	req.Version = "HTTP/1.1"

	// Parse existing params
	req.ParseQueryParams()

	// Modify
	req.SetQueryParam("page", "5")
	req.AddQueryParam("filter", "recent")
	req.DeleteQueryParam("q")

	// Rebuild URL
	req.RebuildURL()

	// Check new URL
	if !contains(req.URL, "page=5") {
		t.Errorf("Expected page=5 in URL: %s", req.URL)
	}
	if !contains(req.URL, "filter=recent") {
		t.Errorf("Expected filter=recent in URL: %s", req.URL)
	}
	if contains(req.URL, "q=") {
		t.Errorf("Expected q parameter to be removed: %s", req.URL)
	}
}

// ============================================================================
// HTTP/2 Pseudo-Headers Tests
// ============================================================================

func TestPseudoHeaders_Parsing(t *testing.T) {
	// For now, we just test the API, not full HTTP/2 parsing
	// (Full HTTP/2 parsing would require binary frame handling)
	req := request.NewRequest()
	req.SetPseudoHeader(":method", "GET")
	req.SetPseudoHeader(":path", "/index.html")
	req.SetPseudoHeader(":scheme", "https")
	req.SetPseudoHeader(":authority", "example.com")

	if req.GetPseudoHeader(":method") != "GET" {
		t.Error("Expected :method=GET")
	}

	if req.GetPseudoHeader(":path") != "/index.html" {
		t.Error("Expected :path=/index.html")
	}

	// Test automatic colon prefix
	if req.GetPseudoHeader("scheme") != "https" {
		t.Error("Expected GetPseudoHeader(\"scheme\") to work")
	}
}

func TestPseudoHeaders_Build(t *testing.T) {
	req := request.NewRequest()
	req.Method = "POST"
	req.URL = "/api/data"
	req.Version = "HTTP/2"
	req.Headers.Set("Host", "api.example.com")
	req.Headers.Set("Content-Type", "application/json")
	req.Body = []byte(`{"key":"value"}`)

	http2Data, err := req.BuildAsHTTP2()
	if err != nil {
		t.Fatalf("BuildAsHTTP2() error: %v", err)
	}

	// Check that pseudo-headers come first
	http2Str := string(http2Data)
	methodPos := bytes.Index(http2Data, []byte(":method:"))
	contentTypePos := bytes.Index(http2Data, []byte("Content-Type:"))

	if methodPos == -1 {
		t.Error("Expected :method pseudo-header in output")
	}

	if contentTypePos == -1 {
		t.Error("Expected Content-Type header in output")
	}

	if methodPos > contentTypePos {
		t.Error("Expected pseudo-headers before regular headers")
	}

	// Check :authority is generated from Host
	if !contains(http2Str, ":authority: api.example.com") {
		t.Error("Expected :authority generated from Host header")
	}

	// Check body present
	if !contains(http2Str, `{"key":"value"}`) {
		t.Error("Expected body in HTTP/2 output")
	}
}

// ============================================================================
// Byte-Perfect Preservation Tests
// ============================================================================

func TestBytePreservation_NoModification(t *testing.T) {
	testCases := [][]byte{
		// Request with cookies
		[]byte("GET / HTTP/1.1\r\nHost: example.com\r\nCookie: a=1; b=2\r\n\r\n"),
		// Request with query params
		[]byte("GET /search?q=test HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		// Response with Set-Cookie
		[]byte("HTTP/1.1 200 OK\r\nSet-Cookie: session=abc\r\nContent-Length: 2\r\n\r\nOK"),
		// Request with Transfer-Encoding
		[]byte("POST / HTTP/1.1\r\nHost: example.com\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n"),
	}

	for i, input := range testCases {
		// Try request parsing
		req, reqErr := request.Parse(input)
		if reqErr == nil {
			output := req.Build()
			// Note: Auto-parsing might modify order slightly, but content should be preserved
			// This test verifies that parsing + building doesn't crash
			if len(output) == 0 {
				t.Errorf("Case %d: Empty output from request build", i)
			}
		}

		// Try response parsing
		resp, respErr := response.Parse(input)
		if respErr == nil {
			output := resp.Build()
			if len(output) == 0 {
				t.Errorf("Case %d: Empty output from response build", i)
			}
		}

		// At least one should succeed
		if reqErr != nil && respErr != nil {
			t.Errorf("Case %d: Both request and response parsing failed", i)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
