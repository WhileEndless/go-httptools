package unit

import (
	"strings"
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/request"
	"github.com/WhileEndless/go-httptools/pkg/response"
)

// ==================== RESPONSE BUILD OPTIONS TESTS ====================

func TestResponse_BuildWithOptions_Default(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: 13\r\n\r\nHello, World!")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	built, err := resp.BuildWithOptions(response.DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}

	if !strings.Contains(string(built), "Hello, World!") {
		t.Error("Build should contain body")
	}
}

func TestResponse_BuildWithOptions_Decompressed(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: 13\r\n\r\nHello, World!")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	built, err := resp.BuildWithOptions(response.DecompressedOptions())
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}

	if !strings.Contains(string(built), "Hello, World!") {
		t.Error("Build should contain body")
	}
}

func TestResponse_BuildWithOptions_HTTP2(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nServer: nginx\r\n\r\n{\"status\":\"ok\"}")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	built, err := resp.BuildWithOptions(response.HTTP2Options())
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}

	builtStr := string(built)

	if !strings.Contains(builtStr, ":status: 200") {
		t.Error("HTTP/2 build should contain :status pseudo-header")
	}

	// Should not contain HTTP/1.x specific headers
	if strings.Contains(builtStr, "Transfer-Encoding") {
		t.Error("HTTP/2 build should not contain Transfer-Encoding")
	}
}

func TestResponse_BuildWithOptions_ChangeCompression(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\nHello, World!")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	opts := response.DefaultBuildOptions()
	opts.Compression = response.CompressionGzip

	built, err := resp.BuildWithOptions(opts)
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}

	builtStr := string(built)

	if !strings.Contains(builtStr, "Content-Encoding: gzip") {
		t.Error("Build should have Content-Encoding: gzip header")
	}
}

func TestResponse_BuildWithOptions_ApplyChunked(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\nHello, World!")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	opts := response.DefaultBuildOptions()
	opts.Chunked = response.ChunkedApply
	opts.ChunkSize = 5

	built, err := resp.BuildWithOptions(opts)
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}

	builtStr := string(built)

	if !strings.Contains(builtStr, "Transfer-Encoding: chunked") {
		t.Error("Build should have Transfer-Encoding: chunked header")
	}

	// Should not have Content-Length
	if strings.Contains(builtStr, "Content-Length") {
		t.Error("Chunked response should not have Content-Length")
	}
}

func TestResponse_BuildWithOptions_NoHeaderUpdate(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: 100\r\n\r\nHello, World!")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	opts := response.DefaultBuildOptions()
	opts.UpdateContentLength = false

	built, err := resp.BuildWithOptions(opts)
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}

	// Content-Length should remain 100 (original value)
	if !strings.Contains(string(built), "Content-Length: 100") {
		t.Error("Content-Length should not be updated when UpdateContentLength=false")
	}
}

func TestResponse_IsCompressed(t *testing.T) {
	resp := response.NewResponse()
	resp.Compressed = true

	if !resp.IsCompressed() {
		t.Error("IsCompressed should return true")
	}
}

func TestResponse_IsChunked(t *testing.T) {
	resp := response.NewResponse()
	resp.IsBodyChunked = true

	if !resp.IsChunked() {
		t.Error("IsChunked should return true")
	}
}

// ==================== REQUEST BUILD OPTIONS TESTS ====================

func TestRequest_BuildWithOptions_Default(t *testing.T) {
	raw := []byte("POST /api HTTP/1.1\r\nHost: example.com\r\nContent-Length: 13\r\n\r\n{\"test\":true}")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	built, err := req.BuildWithOptions(request.DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}

	if !strings.Contains(string(built), "{\"test\":true}") {
		t.Error("Build should contain body")
	}
}

func TestRequest_BuildWithOptions_HTTP2(t *testing.T) {
	raw := []byte("GET /api/users HTTP/1.1\r\nHost: api.example.com\r\nAccept: application/json\r\n\r\n")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	built, err := req.BuildWithOptions(request.HTTP2Options())
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}

	builtStr := string(built)

	if !strings.Contains(builtStr, ":method: GET") {
		t.Error("HTTP/2 build should contain :method pseudo-header")
	}

	if !strings.Contains(builtStr, ":authority: api.example.com") {
		t.Error("HTTP/2 build should contain :authority pseudo-header")
	}

	if !strings.Contains(builtStr, ":path: /api/users") {
		t.Error("HTTP/2 build should contain :path pseudo-header")
	}

	// Host header should not appear in HTTP/2 (it's in :authority)
	lines := strings.Split(builtStr, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "host:") {
			t.Error("HTTP/2 build should not contain Host header")
		}
	}
}

func TestRequest_BuildWithOptions_ChangeCompression(t *testing.T) {
	raw := []byte("POST /api HTTP/1.1\r\nHost: example.com\r\n\r\n{\"data\":\"value\"}")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	opts := request.DefaultBuildOptions()
	opts.Compression = request.CompressionGzip

	built, err := req.BuildWithOptions(opts)
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}

	builtStr := string(built)

	if !strings.Contains(builtStr, "Content-Encoding: gzip") {
		t.Error("Build should have Content-Encoding: gzip header")
	}
}

func TestRequest_BuildWithOptions_ApplyChunked(t *testing.T) {
	raw := []byte("POST /api HTTP/1.1\r\nHost: example.com\r\n\r\n{\"data\":\"value\"}")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	opts := request.DefaultBuildOptions()
	opts.Chunked = request.ChunkedApply

	built, err := req.BuildWithOptions(opts)
	if err != nil {
		t.Fatalf("BuildWithOptions failed: %v", err)
	}

	builtStr := string(built)

	if !strings.Contains(builtStr, "Transfer-Encoding: chunked") {
		t.Error("Build should have Transfer-Encoding: chunked header")
	}
}

func TestRequest_IsCompressed(t *testing.T) {
	req := request.NewRequest()
	req.Compressed = true

	if !req.IsCompressed() {
		t.Error("IsCompressed should return true")
	}
}

func TestRequest_IsChunked(t *testing.T) {
	req := request.NewRequest()
	req.IsBodyChunked = true

	if !req.IsChunked() {
		t.Error("IsChunked should return true")
	}
}

// ==================== CONVENIENCE METHOD TESTS ====================

func TestResponse_BuildNormalized(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\nHello")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	built, err := resp.BuildNormalized()
	if err != nil {
		t.Fatalf("BuildNormalized failed: %v", err)
	}

	if !strings.Contains(string(built), "Hello") {
		t.Error("BuildNormalized should contain body")
	}
}

func TestResponse_BuildAsHTTP2(t *testing.T) {
	raw := []byte("HTTP/1.1 404 Not Found\r\nContent-Type: text/html\r\n\r\n<h1>Not Found</h1>")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	built, err := resp.BuildAsHTTP2()
	if err != nil {
		t.Fatalf("BuildAsHTTP2 failed: %v", err)
	}

	if !strings.Contains(string(built), ":status: 404") {
		t.Error("BuildAsHTTP2 should contain :status pseudo-header")
	}
}

func TestRequest_BuildNormalized(t *testing.T) {
	raw := []byte("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	built, err := req.BuildNormalized()
	if err != nil {
		t.Fatalf("BuildNormalized failed: %v", err)
	}

	if !strings.Contains(string(built), "GET /test HTTP/1.1") {
		t.Error("BuildNormalized should contain request line")
	}
}

func TestRequest_BuildAsHTTP2(t *testing.T) {
	raw := []byte("POST /api/data HTTP/1.1\r\nHost: api.example.com\r\nContent-Type: application/json\r\n\r\n{}")

	req, err := request.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	built, err := req.BuildAsHTTP2()
	if err != nil {
		t.Fatalf("BuildAsHTTP2 failed: %v", err)
	}

	builtStr := string(built)

	if !strings.Contains(builtStr, ":method: POST") {
		t.Error("BuildAsHTTP2 should contain :method pseudo-header")
	}
}
