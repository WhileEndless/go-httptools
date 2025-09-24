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
