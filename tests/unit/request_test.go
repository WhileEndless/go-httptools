package unit

import (
	"bytes"
	"io"
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

	// Get returns original values (with whitespace preserved)
	if got := req.Headers.Get("Host"); got != "  example.com  " {
		t.Errorf("Get('Host') expected '  example.com  ', got '%s'", got)
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

	// Verify parsed values (original, untrimmed)
	expectations := map[string]string{
		"Host":       "example.com",
		"User-Agent": "  Mozilla  ",
		"Accept":     "\t*/*",
		"X-Empty":    "",
		"X-Spaces":   "   value   ",
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

// ============================================================================
// ParseReader Tests
// ============================================================================

func TestRequestParseReader_Basic(t *testing.T) {
	raw := []byte(`GET /api/users HTTP/1.1
Host: example.com
User-Agent: test

`)

	reader := bytes.NewReader(raw)
	req, err := request.ParseReader(reader)
	if err != nil {
		t.Fatalf("ParseReader failed: %v", err)
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
}

func TestRequestParseReader_WithBody(t *testing.T) {
	raw := []byte(`POST /api/login HTTP/1.1
Host: example.com
Content-Type: application/json

{"username":"admin","password":"pass"}`)

	reader := bytes.NewReader(raw)
	req, err := request.ParseReader(reader)
	if err != nil {
		t.Fatalf("ParseReader failed: %v", err)
	}

	expectedBody := `{"username":"admin","password":"pass"}`
	if string(req.Body) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(req.Body))
	}
}

// ============================================================================
// Streaming Support Tests (ParseHeadersFromReader, WriteTo)
// ============================================================================

func TestRequestParseHeadersFromReader_Basic(t *testing.T) {
	raw := "POST /api/upload HTTP/1.1\r\nHost: example.com\r\nContent-Length: 1000\r\n\r\ntest body data"

	reader := bytes.NewReader([]byte(raw))
	req, bodyReader, err := request.ParseHeadersFromReader(reader)
	if err != nil {
		t.Fatalf("ParseHeadersFromReader failed: %v", err)
	}

	// Check headers are parsed
	if req.Method != "POST" {
		t.Errorf("Expected method POST, got %s", req.Method)
	}

	if req.URL != "/api/upload" {
		t.Errorf("Expected URL /api/upload, got %s", req.URL)
	}

	if req.GetHost() != "example.com" {
		t.Errorf("Expected Host example.com, got %s", req.GetHost())
	}

	// Check body is NOT read yet (should be in bodyReader)
	if len(req.Body) != 0 {
		t.Errorf("Expected empty body in request, got %d bytes", len(req.Body))
	}

	// Read body from bodyReader
	bodyData, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	expectedBody := "test body data"
	if string(bodyData) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(bodyData))
	}
}

func TestRequestWriteTo_Basic(t *testing.T) {
	req := request.NewRequest()
	req.Method = "POST"
	req.URL = "/api/data"
	req.Version = "HTTP/1.1"
	req.LineSeparator = "\r\n"
	req.Headers.Set("Host", "example.com")
	req.Headers.Set("Content-Type", "application/json")
	req.Body = []byte(`{"test":"data"}`)

	var buf bytes.Buffer
	n, err := req.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if n != int64(buf.Len()) {
		t.Errorf("Bytes written mismatch: returned %d, actual %d", n, buf.Len())
	}

	// Verify the output can be parsed back
	parsed, err := request.Parse(buf.Bytes())
	if err != nil {
		t.Fatalf("Failed to parse written request: %v", err)
	}

	if parsed.Method != "POST" {
		t.Errorf("Expected method POST, got %s", parsed.Method)
	}

	if parsed.URL != "/api/data" {
		t.Errorf("Expected URL /api/data, got %s", parsed.URL)
	}

	if string(parsed.Body) != `{"test":"data"}` {
		t.Errorf("Body mismatch after WriteTo")
	}
}

func TestRequestWriteHeadersTo_Basic(t *testing.T) {
	req := request.NewRequest()
	req.Method = "GET"
	req.URL = "/test"
	req.Version = "HTTP/1.1"
	req.LineSeparator = "\r\n"
	req.Headers.Set("Host", "example.com")

	var buf bytes.Buffer
	n, err := req.WriteHeadersTo(&buf)
	if err != nil {
		t.Fatalf("WriteHeadersTo failed: %v", err)
	}

	if n != int64(buf.Len()) {
		t.Errorf("Bytes written mismatch")
	}

	output := buf.String()
	if !bytes.HasPrefix(buf.Bytes(), []byte("GET /test HTTP/1.1\r\n")) {
		t.Errorf("Request line not correct: %s", output)
	}

	if !bytes.Contains(buf.Bytes(), []byte("Host: example.com")) {
		t.Errorf("Host header missing")
	}
}

func TestRequestStreamingRoundTrip(t *testing.T) {
	// Create a request, write it to a buffer, then parse headers from it
	originalReq := request.NewRequest()
	originalReq.Method = "PUT"
	originalReq.URL = "/api/file"
	originalReq.Version = "HTTP/1.1"
	originalReq.LineSeparator = "\r\n"
	originalReq.Headers.Set("Host", "example.com")
	originalReq.Headers.Set("Content-Type", "application/octet-stream")
	originalReq.Headers.Set("Content-Length", "50")
	originalReq.Body = bytes.Repeat([]byte("B"), 50)

	// Write to buffer
	var buf bytes.Buffer
	_, err := originalReq.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	// Parse headers from buffer
	parsedReq, bodyReader, err := request.ParseHeadersFromReader(&buf)
	if err != nil {
		t.Fatalf("ParseHeadersFromReader failed: %v", err)
	}

	// Verify headers
	if parsedReq.Method != originalReq.Method {
		t.Errorf("Method mismatch: expected %s, got %s", originalReq.Method, parsedReq.Method)
	}

	if parsedReq.URL != originalReq.URL {
		t.Errorf("URL mismatch: expected %s, got %s", originalReq.URL, parsedReq.URL)
	}

	if parsedReq.GetHost() != "example.com" {
		t.Errorf("Host mismatch")
	}

	// Stream body
	bodyData, _ := io.ReadAll(bodyReader)
	if len(bodyData) != 50 {
		t.Errorf("Body length mismatch: expected 50, got %d", len(bodyData))
	}
}

func TestRequestParseReader_EmptyReader(t *testing.T) {
	reader := bytes.NewReader([]byte{})
	_, err := request.ParseReader(reader)
	if err == nil {
		t.Error("Expected error for empty reader")
	}
}

// requestErrorReader simulates a reader that returns an error
type requestErrorReader struct{}

func (e *requestErrorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestRequestParseReader_ReaderError(t *testing.T) {
	reader := &requestErrorReader{}
	_, err := request.ParseReader(reader)
	if err == nil {
		t.Error("Expected error when reader fails")
	}
}
