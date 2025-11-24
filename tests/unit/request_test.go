package unit

import (
	"bytes"
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/request"
)

func TestRequestParse_Basic(t *testing.T) {
	raw := []byte(`GET /api/users HTTP/1.1
Host: example.com
User-Agent: test
test:deneme

`)

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Expected method GET, got %s", req.Method)
	}

	if req.URL != "/api/users" {
		t.Errorf("Expected URL /api/users, got %s", req.URL)
	}

	if req.Version != "HTTP/1.1" {
		t.Errorf("Expected version HTTP/1.1, got %s", req.Version)
	}

	if got := req.Headers.Get("test"); got != "deneme" {
		t.Errorf("Expected test header 'deneme', got '%s'", got)
	}
}

func TestRequestParse_WithBody(t *testing.T) {
	raw := []byte(`POST /api/login HTTP/1.1
Host: example.com
Content-Type: application/json
test:deneme

{"username":"admin","password":"pass"}`)

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expectedBody := `{"username":"admin","password":"pass"}`
	if string(req.Body) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(req.Body))
	}
}

func TestRequestParse_FaultTolerance(t *testing.T) {
	// Malformed request with missing version
	raw := []byte(`GET /path
Host: example.com
: empty-header-name
Invalid-Header-No-Colon
test:deneme

`)

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Fault tolerant parse should succeed: %v", err)
	}

	// Should default to HTTP/1.1
	if req.Version != "HTTP/1.1" {
		t.Errorf("Expected default version HTTP/1.1, got %s", req.Version)
	}

	// Should have parsed normal headers
	if got := req.Headers.Get("test"); got != "deneme" {
		t.Errorf("Expected test header 'deneme', got '%s'", got)
	}
}

func TestRequestParse_MinimalRequest(t *testing.T) {
	// Absolute minimal request
	raw := []byte(`GET /`)

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse minimal request failed: %v", err)
	}

	if req.Method != "GET" || req.URL != "/" {
		t.Errorf("Minimal parse failed: method=%s, url=%s", req.Method, req.URL)
	}

	if req.Version != "HTTP/1.1" {
		t.Errorf("Expected default version, got %s", req.Version)
	}
}

func TestRequestBuild_Reconstruction(t *testing.T) {
	raw := []byte(`POST /api/test HTTP/1.1
Host: example.com
Content-Type: application/json
test:deneme
Authorization: Bearer token

{"data":"test"}`)

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	rebuilt := req.Build()

	// Parse rebuilt request
	req2, err := request.Parse(rebuilt)
	if err != nil {
		t.Fatalf("Rebuild parse failed: %v", err)
	}

	// Should be identical
	if req.Method != req2.Method {
		t.Errorf("Method mismatch after rebuild")
	}

	if req.URL != req2.URL {
		t.Errorf("URL mismatch after rebuild")
	}

	if !bytes.Equal(req.Body, req2.Body) {
		t.Errorf("Body mismatch after rebuild")
	}

	// Check header order preservation
	headers1 := req.Headers.All()
	headers2 := req2.Headers.All()

	if len(headers1) != len(headers2) {
		t.Errorf("Header count mismatch: %d vs %d", len(headers1), len(headers2))
	}

	for i, h1 := range headers1 {
		h2 := headers2[i]
		if h1.Name != h2.Name || h1.Value != h2.Value {
			t.Errorf("Header order not preserved at position %d", i)
		}
	}
}

func TestRequestClone(t *testing.T) {
	raw := []byte(`GET /test HTTP/1.1
Host: example.com
test:deneme

body content`)

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	clone := req.Clone()

	// Modify original
	req.Method = "POST"
	req.Headers.Set("New-Header", "value")
	req.Body = []byte("new body")

	// Clone should be unchanged
	if clone.Method != "GET" {
		t.Errorf("Clone modified when original changed")
	}

	if clone.Headers.Has("New-Header") {
		t.Errorf("Clone headers modified when original changed")
	}

	if string(clone.Body) == "new body" {
		t.Errorf("Clone body modified when original changed")
	}
}

func TestRequestUtilityMethods(t *testing.T) {
	raw := []byte(`GET /test HTTP/1.1
Host: example.com
User-Agent: TestAgent
Content-Type: application/json
test:deneme

`)

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if req.GetHost() != "example.com" {
		t.Errorf("GetHost failed")
	}

	if req.GetUserAgent() != "TestAgent" {
		t.Errorf("GetUserAgent failed")
	}

	if req.GetContentType() != "application/json" {
		t.Errorf("GetContentType failed")
	}
}

func TestRequestSetBody(t *testing.T) {
	req := request.NewRequest()
	req.Method = "POST"
	req.URL = "/test"

	body := []byte(`{"test":"data"}`)
	req.SetBody(body)

	if !bytes.Equal(req.Body, body) {
		t.Errorf("SetBody failed")
	}

	if req.GetContentLength() != "15" {
		t.Errorf("Content-Length not set correctly: got %s", req.GetContentLength())
	}
}

// ==================== REQUEST FORMAT PRESERVATION TESTS ====================

func TestRequestParse_PreserveHeaderFormat(t *testing.T) {
	// Test that header formatting is preserved through request parsing
	raw := []byte("GET / HTTP/1.1\r\nHost:  example.com  \r\nX-Custom:value\r\n\r\n")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Get should return trimmed values
	if got := req.Headers.Get("Host"); got != "example.com" {
		t.Errorf("Get('Host') expected 'example.com', got '%s'", got)
	}

	if got := req.Headers.Get("X-Custom"); got != "value" {
		t.Errorf("Get('X-Custom') expected 'value', got '%s'", got)
	}

	// Build headers should preserve format
	headerBytes := req.Headers.Build()
	expectedHeaders := "Host:  example.com  \r\nX-Custom:value\r\n"
	if string(headerBytes) != expectedHeaders {
		t.Errorf("Header format not preserved:\nExpected: %q\nGot: %q", expectedHeaders, headerBytes)
	}
}

func TestRequestParse_PreserveLineEndings(t *testing.T) {
	// Test LF only line endings
	raw := []byte("GET / HTTP/1.1\nHost: example.com\nUser-Agent: test\n\n")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	headerBytes := req.Headers.Build()
	expectedHeaders := "Host: example.com\nUser-Agent: test\n"
	if string(headerBytes) != expectedHeaders {
		t.Errorf("LF line endings not preserved:\nExpected: %q\nGot: %q", expectedHeaders, headerBytes)
	}
}

func TestRequestParse_MixedLineEndings(t *testing.T) {
	// Test mixed line endings
	raw := []byte("GET / HTTP/1.1\r\nHost: example.com\nX-Custom: value\r\n\r\n")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	headerBytes := req.Headers.Build()
	expectedHeaders := "Host: example.com\nX-Custom: value\r\n"
	if string(headerBytes) != expectedHeaders {
		t.Errorf("Mixed line endings not preserved:\nExpected: %q\nGot: %q", expectedHeaders, headerBytes)
	}
}

func TestRequestParse_DoubleCarriageReturn(t *testing.T) {
	// Test \r\r\n edge case
	raw := []byte("GET / HTTP/1.1\r\nHost: example.com\r\r\n\r\n")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	headerBytes := req.Headers.Build()
	expectedHeaders := "Host: example.com\r\r\n"
	if string(headerBytes) != expectedHeaders {
		t.Errorf("Double CR not preserved:\nExpected: %q\nGot: %q", expectedHeaders, headerBytes)
	}
}

func TestRequestBuild_PreservesOriginalHeaderFormat(t *testing.T) {
	// Parse request with non-standard header formatting
	raw := []byte("GET /test HTTP/1.1\r\nHost:  example.com  \r\nX-Custom:no-space\r\nX-Tab:\ttabbed\r\n\r\nbody")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Rebuild and check headers
	rebuilt := req.Build()

	// Parse rebuilt to check headers
	req2, err := request.Parse(rebuilt)
	if err != nil {
		t.Fatalf("Parse rebuilt failed: %v", err)
	}

	// Original headers should match
	originalHeaders := req.Headers.All()
	rebuiltHeaders := req2.Headers.All()

	if len(originalHeaders) != len(rebuiltHeaders) {
		t.Fatalf("Header count mismatch: %d vs %d", len(originalHeaders), len(rebuiltHeaders))
	}

	for i, orig := range originalHeaders {
		rebuilt := rebuiltHeaders[i]
		if orig.Name != rebuilt.Name {
			t.Errorf("Header name mismatch at %d: %s vs %s", i, orig.Name, rebuilt.Name)
		}
		if orig.Value != rebuilt.Value {
			t.Errorf("Header value mismatch at %d: %s vs %s", i, orig.Value, rebuilt.Value)
		}
		// OriginalLine should match
		if orig.OriginalLine != rebuilt.OriginalLine {
			t.Errorf("OriginalLine mismatch at %d: %q vs %q", i, orig.OriginalLine, rebuilt.OriginalLine)
		}
	}
}

func TestRequestParse_ModifyHeaderClearsFormat(t *testing.T) {
	// When modifying a header, original format should be cleared
	raw := []byte("GET / HTTP/1.1\r\nHost:  example.com  \r\n\r\n")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Modify the Host header
	req.Headers.Set("Host", "newhost.com")

	// Build should use standard format for modified header
	headerBytes := req.Headers.Build()
	expected := "Host: newhost.com\r\n"
	if string(headerBytes) != expected {
		t.Errorf("Modified header should use standard format:\nExpected: %q\nGot: %q", expected, headerBytes)
	}
}

func TestRequestParse_AddNewHeaderUsesStandardFormat(t *testing.T) {
	raw := []byte("GET / HTTP/1.1\r\nHost:  example.com  \r\n\r\n")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Add new header
	req.Headers.Set("X-New", "value")

	headerBytes := req.Headers.Build()
	expected := "Host:  example.com  \r\nX-New: value\r\n"
	if string(headerBytes) != expected {
		t.Errorf("New header should use standard format:\nExpected: %q\nGot: %q", expected, headerBytes)
	}
}

func TestRequestParse_ComplexFormatPreservation(t *testing.T) {
	// Complex test with various formatting quirks
	raw := []byte("GET /path?query=1 HTTP/1.1\r\n" +
		"Host:example.com\r\n" +          // No space
		"User-Agent:  Mozilla  \r\n" +     // Double space
		"Accept:\t*/*\r\n" +               // Tab
		"X-Empty:\r\n" +                   // Empty value
		"X-Spaces:   value   \r\n" +       // Multiple spaces
		"\r\n" +
		"body content")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify parsed values (trimmed)
	expectations := map[string]string{
		"Host":       "example.com",
		"User-Agent": "Mozilla",
		"Accept":     "*/*",
		"X-Empty":    "",
		"X-Spaces":   "value",
	}

	for name, expected := range expectations {
		if got := req.Headers.Get(name); got != expected {
			t.Errorf("Get(%s) expected %q, got %q", name, expected, got)
		}
	}

	// Verify original format preserved
	headerBytes := req.Headers.Build()
	expectedHeaders := "Host:example.com\r\n" +
		"User-Agent:  Mozilla  \r\n" +
		"Accept:\t*/*\r\n" +
		"X-Empty:\r\n" +
		"X-Spaces:   value   \r\n"

	if string(headerBytes) != expectedHeaders {
		t.Errorf("Complex format not preserved:\nExpected: %q\nGot: %q", expectedHeaders, headerBytes)
	}
}
