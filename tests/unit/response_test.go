package unit

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/WhileEndless/go-httptools/pkg/response"
	"github.com/andybalholm/brotli"
)

func TestResponseParse_Basic(t *testing.T) {
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
Server: nginx/1.18.0
test:deneme

{"message":"success"}`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if resp.Version != "HTTP/1.1" {
		t.Errorf("Expected version HTTP/1.1, got %s", resp.Version)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.StatusText != "OK" {
		t.Errorf("Expected status text OK, got %s", resp.StatusText)
	}

	if got := resp.Headers.Get("test"); got != "deneme" {
		t.Errorf("Expected test header 'deneme', got '%s'", got)
	}

	expectedBody := `{"message":"success"}`
	if string(resp.Body) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(resp.Body))
	}
}

func TestResponseParse_FaultTolerance(t *testing.T) {
	// Invalid status code
	raw := []byte(`HTTP/1.1 999 Custom Status
Content-Type: text/plain
test:deneme

Test content`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Fault tolerant parse should succeed: %v", err)
	}

	if resp.StatusCode != 999 {
		t.Errorf("Expected status 999, got %d", resp.StatusCode)
	}

	if resp.StatusText != "Custom Status" {
		t.Errorf("Expected custom status text, got %s", resp.StatusText)
	}
}

func TestResponseParse_DefaultStatusText(t *testing.T) {
	// Missing status text
	raw := []byte(`HTTP/1.1 404
Content-Type: text/html

Not found`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if resp.StatusText != "Not Found" {
		t.Errorf("Expected default status text 'Not Found', got '%s'", resp.StatusText)
	}
}

func TestResponseBuild_Reconstruction(t *testing.T) {
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
Server: test-server
test:deneme

{"data":"value"}`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	rebuilt := resp.Build()

	// Parse rebuilt response
	resp2, err := response.Parse(rebuilt)
	if err != nil {
		t.Fatalf("Rebuild parse failed: %v", err)
	}

	// Should be identical
	if resp.StatusCode != resp2.StatusCode {
		t.Errorf("Status code mismatch after rebuild")
	}

	if resp.StatusText != resp2.StatusText {
		t.Errorf("Status text mismatch after rebuild")
	}

	if !bytes.Equal(resp.Body, resp2.Body) {
		t.Errorf("Body mismatch after rebuild")
	}
}

func TestResponseStatusMethods(t *testing.T) {
	tests := []struct {
		status     int
		successful bool
		redirect   bool
		clientErr  bool
		serverErr  bool
	}{
		{200, true, false, false, false},
		{201, true, false, false, false},
		{301, false, true, false, false},
		{404, false, false, true, false},
		{500, false, false, false, true},
	}

	for _, test := range tests {
		resp := response.NewResponse()
		resp.StatusCode = test.status

		if resp.IsSuccessful() != test.successful {
			t.Errorf("Status %d: IsSuccessful() = %t, expected %t",
				test.status, resp.IsSuccessful(), test.successful)
		}

		if resp.IsRedirect() != test.redirect {
			t.Errorf("Status %d: IsRedirect() = %t, expected %t",
				test.status, resp.IsRedirect(), test.redirect)
		}

		if resp.IsClientError() != test.clientErr {
			t.Errorf("Status %d: IsClientError() = %t, expected %t",
				test.status, resp.IsClientError(), test.clientErr)
		}

		if resp.IsServerError() != test.serverErr {
			t.Errorf("Status %d: IsServerError() = %t, expected %t",
				test.status, resp.IsServerError(), test.serverErr)
		}
	}
}

func TestResponseClone(t *testing.T) {
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
test:deneme

{"original":"data"}`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	clone := resp.Clone()

	// Modify original
	resp.StatusCode = 404
	resp.Headers.Set("New-Header", "value")
	resp.Body = []byte("modified")

	// Clone should be unchanged
	if clone.StatusCode != 200 {
		t.Errorf("Clone modified when original changed")
	}

	if clone.Headers.Has("New-Header") {
		t.Errorf("Clone headers modified when original changed")
	}

	if string(clone.Body) == "modified" {
		t.Errorf("Clone body modified when original changed")
	}
}

func TestResponseUtilityMethods(t *testing.T) {
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 15
Server: TestServer
test:deneme

{"test":"data"}`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if resp.GetContentType() != "application/json" {
		t.Errorf("GetContentType failed")
	}

	if resp.GetContentLength() != 15 {
		t.Errorf("GetContentLength failed: got %d", resp.GetContentLength())
	}

	if resp.GetServer() != "TestServer" {
		t.Errorf("GetServer failed")
	}
}

func TestResponseRedirection(t *testing.T) {
	raw := []byte(`HTTP/1.1 302 Found
Location: https://example.com/new-path
test:deneme

`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !resp.IsRedirect() {
		t.Errorf("Should be redirect")
	}

	location := resp.GetRedirectLocation()
	if location != "https://example.com/new-path" {
		t.Errorf("Expected redirect location, got '%s'", location)
	}
}

// ============================================================================
// Chunked Transfer Encoding Tests
// ============================================================================

func TestResponseParse_ChunkedDefault(t *testing.T) {
	// Default behavior: chunked body is NOT auto-decoded
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
Transfer-Encoding: chunked

5
hello
5
world
0

`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should detect chunked encoding
	if !resp.IsBodyChunked {
		t.Error("Expected IsBodyChunked=true")
	}

	// Body should still be chunked (not decoded by default)
	if !bytes.Contains(resp.Body, []byte("5\nhello")) {
		t.Error("Expected body to remain chunked by default")
	}

	// Transfer-Encoding header should be present (value may have leading space)
	if strings.TrimSpace(resp.Headers.Get("Transfer-Encoding")) != "chunked" {
		t.Error("Expected Transfer-Encoding header to be preserved")
	}
}

func TestResponseParseWithOptions_AutoDecodeChunked(t *testing.T) {
	// With AutoDecodeChunked: body is automatically decoded
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
Transfer-Encoding: chunked

5
hello
5
world
0

`)

	opts := response.ParseOptions{
		AutoDecodeChunked: true,
	}

	resp, err := response.ParseWithOptions(raw, opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have decoded the body
	expectedBody := "helloworld"
	if string(resp.Body) != expectedBody {
		t.Errorf("Expected decoded body '%s', got '%s'", expectedBody, string(resp.Body))
	}

	// IsBodyChunked should be false after decoding
	if resp.IsBodyChunked {
		t.Error("Expected IsBodyChunked=false after auto-decode")
	}

	// Transfer-Encoding header should be removed
	if resp.Headers.Get("Transfer-Encoding") != "" {
		t.Error("Expected Transfer-Encoding header to be removed after decoding")
	}

	// Content-Length should be added
	contentLength := resp.Headers.Get("Content-Length")
	if contentLength != "10" {
		t.Errorf("Expected Content-Length=10, got '%s'", contentLength)
	}

	// RawBody should contain original chunked data
	if !bytes.Contains(resp.RawBody, []byte("5\nhello")) {
		t.Error("Expected RawBody to contain original chunked data")
	}
}

func TestResponseParseWithOptions_ChunkedWithTrailers(t *testing.T) {
	// Note: Proper chunked encoding uses \r\n
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\nX-Checksum: abc123\r\nX-Custom: value\r\n\r\n")

	opts := response.ParseOptions{
		AutoDecodeChunked:       true,
		PreserveChunkedTrailers: true,
	}

	resp, err := response.ParseWithOptions(raw, opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Body should be decoded
	if string(resp.Body) != "hello" {
		t.Errorf("Expected decoded body 'hello', got '%s'", string(resp.Body))
	}

	// Trailers should be preserved as headers
	if resp.Headers.Get("X-Checksum") != "abc123" {
		t.Error("Expected trailer X-Checksum to be preserved as header")
	}

	if resp.Headers.Get("X-Custom") != "value" {
		t.Error("Expected trailer X-Custom to be preserved as header")
	}
}

func TestResponseParseWithOptions_ChunkedWithoutTrailerPreservation(t *testing.T) {
	raw := []byte(`HTTP/1.1 200 OK
Transfer-Encoding: chunked

5
hello
0
X-Checksum: abc123

`)

	opts := response.ParseOptions{
		AutoDecodeChunked:       true,
		PreserveChunkedTrailers: false, // Don't preserve trailers
	}

	resp, err := response.ParseWithOptions(raw, opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Body should be decoded
	if string(resp.Body) != "hello" {
		t.Errorf("Expected decoded body 'hello', got '%s'", string(resp.Body))
	}

	// Trailers should NOT be preserved as headers
	if resp.Headers.Get("X-Checksum") != "" {
		t.Error("Expected trailer X-Checksum NOT to be preserved as header")
	}
}

func TestResponseParseWithOptions_NonChunkedResponse(t *testing.T) {
	// Regular response without chunked encoding
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 5

hello`)

	opts := response.ParseOptions{
		AutoDecodeChunked: true, // Should have no effect
	}

	resp, err := response.ParseWithOptions(raw, opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Body should be unchanged
	if string(resp.Body) != "hello" {
		t.Errorf("Expected body 'hello', got '%s'", string(resp.Body))
	}

	// Should not be marked as chunked
	if resp.IsBodyChunked {
		t.Error("Expected IsBodyChunked=false for non-chunked response")
	}
}

func TestResponseParseWithOptions_EmptyChunkedBody(t *testing.T) {
	raw := []byte(`HTTP/1.1 200 OK
Transfer-Encoding: chunked

0

`)

	opts := response.ParseOptions{
		AutoDecodeChunked: true,
	}

	resp, err := response.ParseWithOptions(raw, opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Body should be empty
	if len(resp.Body) != 0 {
		t.Errorf("Expected empty body, got '%s'", string(resp.Body))
	}

	// Content-Length should not be set for empty body
	if resp.Headers.Get("Content-Length") != "" {
		t.Error("Expected no Content-Length for empty decoded body")
	}
}

func TestResponseParseWithOptions_ComplexChunked(t *testing.T) {
	// Note: Proper chunked encoding uses \r\n
	// First chunk: 28 bytes (0x1c), second chunk: 26 bytes (0x1a)
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nTransfer-Encoding: chunked\r\n\r\n1c\r\n{\"message\":\"This is a test\"}\r\n1a\r\n{\"additional\":\"data here\"}\r\n0\r\n\r\n")

	opts := response.ParseOptions{
		AutoDecodeChunked: true,
	}

	resp, err := response.ParseWithOptions(raw, opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expectedBody := `{"message":"This is a test"}{"additional":"data here"}`
	actualBody := string(resp.Body)
	if actualBody != expectedBody {
		t.Errorf("Expected decoded body '%s' (len=%d), got '%s' (len=%d)",
			expectedBody, len(expectedBody), actualBody, len(actualBody))
		t.Errorf("Body bytes: %v", resp.Body)
	}

	// Verify Content-Length matches actual body length
	expectedLength := len(actualBody)
	actualLength := resp.GetContentLength()
	if actualLength != expectedLength {
		t.Errorf("Expected Content-Length=%d (body length), got %d", expectedLength, actualLength)
	}
}

// ============================================================================
// Content Encoding / Compression Tests
// ============================================================================

func TestResponseParse_GzipDecompression(t *testing.T) {
	// Manually compress body with gzip
	originalBody := []byte(`{"message":"This is compressed with gzip!"}`)

	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	if _, err := gzipWriter.Write(originalBody); err != nil {
		t.Fatalf("Failed to compress with gzip: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	// Build HTTP response with gzip-compressed body
	raw := bytes.Buffer{}
	raw.WriteString("HTTP/1.1 200 OK\r\n")
	raw.WriteString("Content-Type: application/json\r\n")
	raw.WriteString("Content-Encoding: gzip\r\n")
	raw.WriteString("\r\n")
	raw.Write(gzipBuf.Bytes())

	resp, err := response.Parse(raw.Bytes())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should automatically decompress
	if !resp.Compressed {
		t.Error("Expected Compressed=true for gzip response")
	}

	if !bytes.Equal(resp.Body, originalBody) {
		t.Errorf("Body not decompressed correctly.\nExpected: %s\nGot: %s",
			string(originalBody), string(resp.Body))
	}

	// RawBody should contain compressed data
	if !bytes.Equal(resp.RawBody, gzipBuf.Bytes()) {
		t.Error("RawBody should contain original compressed data")
	}
}

func TestResponseParse_DeflateDecompression(t *testing.T) {
	// Manually compress body with deflate
	originalBody := []byte(`{"message":"This is compressed with deflate!"}`)

	var deflateBuf bytes.Buffer
	deflateWriter, err := flate.NewWriter(&deflateBuf, flate.DefaultCompression)
	if err != nil {
		t.Fatalf("Failed to create deflate writer: %v", err)
	}
	if _, err := deflateWriter.Write(originalBody); err != nil {
		t.Fatalf("Failed to compress with deflate: %v", err)
	}
	if err := deflateWriter.Close(); err != nil {
		t.Fatalf("Failed to close deflate writer: %v", err)
	}

	// Build HTTP response with deflate-compressed body
	raw := bytes.Buffer{}
	raw.WriteString("HTTP/1.1 200 OK\r\n")
	raw.WriteString("Content-Type: application/json\r\n")
	raw.WriteString("Content-Encoding: deflate\r\n")
	raw.WriteString("\r\n")
	raw.Write(deflateBuf.Bytes())

	resp, err := response.Parse(raw.Bytes())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should automatically decompress
	if !resp.Compressed {
		t.Error("Expected Compressed=true for deflate response")
	}

	if !bytes.Equal(resp.Body, originalBody) {
		t.Errorf("Body not decompressed correctly.\nExpected: %s\nGot: %s",
			string(originalBody), string(resp.Body))
	}

	// RawBody should contain compressed data
	if !bytes.Equal(resp.RawBody, deflateBuf.Bytes()) {
		t.Error("RawBody should contain original compressed data")
	}
}

func TestResponseParse_BrotliDecompression(t *testing.T) {
	// Manually compress body with brotli
	originalBody := []byte(`{"message":"This is compressed with brotli!"}`)

	var brotliBuf bytes.Buffer
	brotliWriter := brotli.NewWriter(&brotliBuf)
	if _, err := brotliWriter.Write(originalBody); err != nil {
		t.Fatalf("Failed to compress with brotli: %v", err)
	}
	if err := brotliWriter.Close(); err != nil {
		t.Fatalf("Failed to close brotli writer: %v", err)
	}

	// Build HTTP response with brotli-compressed body
	raw := bytes.Buffer{}
	raw.WriteString("HTTP/1.1 200 OK\r\n")
	raw.WriteString("Content-Type: application/json\r\n")
	raw.WriteString("Content-Encoding: br\r\n")
	raw.WriteString("\r\n")
	raw.Write(brotliBuf.Bytes())

	resp, err := response.Parse(raw.Bytes())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should automatically decompress
	if !resp.Compressed {
		t.Error("Expected Compressed=true for brotli response")
	}

	if !bytes.Equal(resp.Body, originalBody) {
		t.Errorf("Body not decompressed correctly.\nExpected: %s\nGot: %s",
			string(originalBody), string(resp.Body))
	}

	// RawBody should contain compressed data
	if !bytes.Equal(resp.RawBody, brotliBuf.Bytes()) {
		t.Error("RawBody should contain original compressed data")
	}
}

func TestResponseParse_BrotliFullName(t *testing.T) {
	// Test with "brotli" instead of "br"
	originalBody := []byte(`{"message":"Testing brotli full name encoding!"}`)

	var brotliBuf bytes.Buffer
	brotliWriter := brotli.NewWriter(&brotliBuf)
	if _, err := brotliWriter.Write(originalBody); err != nil {
		t.Fatalf("Failed to compress with brotli: %v", err)
	}
	if err := brotliWriter.Close(); err != nil {
		t.Fatalf("Failed to close brotli writer: %v", err)
	}

	// Build HTTP response with "brotli" content encoding
	raw := bytes.Buffer{}
	raw.WriteString("HTTP/1.1 200 OK\r\n")
	raw.WriteString("Content-Type: application/json\r\n")
	raw.WriteString("Content-Encoding: brotli\r\n")
	raw.WriteString("\r\n")
	raw.Write(brotliBuf.Bytes())

	resp, err := response.Parse(raw.Bytes())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should automatically decompress
	if !resp.Compressed {
		t.Error("Expected Compressed=true for brotli response")
	}

	if !bytes.Equal(resp.Body, originalBody) {
		t.Errorf("Body not decompressed correctly.\nExpected: %s\nGot: %s",
			string(originalBody), string(resp.Body))
	}
}

func TestResponseParse_NoCompressionHeader(t *testing.T) {
	// Response without Content-Encoding header
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: text/plain

Hello, World!`)

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should not be marked as compressed
	if resp.Compressed {
		t.Error("Expected Compressed=false for uncompressed response")
	}

	expectedBody := "Hello, World!"
	if string(resp.Body) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(resp.Body))
	}
}

func TestResponseParse_InvalidCompressedData(t *testing.T) {
	// Invalid gzip data - should fall back to raw body (fault tolerance)
	invalidGzipData := []byte("This is not valid gzip data!")

	raw := bytes.Buffer{}
	raw.WriteString("HTTP/1.1 200 OK\r\n")
	raw.WriteString("Content-Encoding: gzip\r\n")
	raw.WriteString("\r\n")
	raw.Write(invalidGzipData)

	resp, err := response.Parse(raw.Bytes())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should not be marked as compressed (decompression failed)
	if resp.Compressed {
		t.Error("Expected Compressed=false when decompression fails")
	}

	// Body should contain the raw (invalid) data
	if !bytes.Equal(resp.Body, invalidGzipData) {
		t.Error("Expected Body to contain raw data when decompression fails")
	}
}

func TestResponseParse_LargeBrotliContent(t *testing.T) {
	// Test with larger content
	originalBody := bytes.Repeat([]byte("This is a repeated message for testing brotli compression with larger payloads. "), 100)

	var brotliBuf bytes.Buffer
	brotliWriter := brotli.NewWriter(&brotliBuf)
	if _, err := brotliWriter.Write(originalBody); err != nil {
		t.Fatalf("Failed to compress with brotli: %v", err)
	}
	if err := brotliWriter.Close(); err != nil {
		t.Fatalf("Failed to close brotli writer: %v", err)
	}

	// Build HTTP response
	raw := bytes.Buffer{}
	raw.WriteString("HTTP/1.1 200 OK\r\n")
	raw.WriteString("Content-Type: text/plain\r\n")
	raw.WriteString("Content-Encoding: br\r\n")
	raw.WriteString("\r\n")
	raw.Write(brotliBuf.Bytes())

	resp, err := response.Parse(raw.Bytes())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify decompression
	if !resp.Compressed {
		t.Error("Expected Compressed=true")
	}

	if !bytes.Equal(resp.Body, originalBody) {
		t.Errorf("Body not decompressed correctly. Expected %d bytes, got %d bytes",
			len(originalBody), len(resp.Body))
	}

	// Verify compression is actually happening (compressed should be smaller)
	compressionRatio := float64(len(brotliBuf.Bytes())) / float64(len(originalBody))
	if compressionRatio >= 0.9 {
		t.Logf("Warning: Compression ratio is %.2f%% - might not be compressing effectively",
			compressionRatio*100)
	}
}

func TestResponseParse_HTMLWithBrotli(t *testing.T) {
	// Real-world scenario: HTML compressed with brotli
	originalHTML := []byte(`<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
    <meta charset="UTF-8">
</head>
<body>
    <h1>Welcome to the test page!</h1>
    <p>This HTML content is compressed with Brotli compression.</p>
    <ul>
        <li>Item 1</li>
        <li>Item 2</li>
        <li>Item 3</li>
    </ul>
</body>
</html>`)

	var brotliBuf bytes.Buffer
	brotliWriter := brotli.NewWriter(&brotliBuf)
	if _, err := brotliWriter.Write(originalHTML); err != nil {
		t.Fatalf("Failed to compress HTML: %v", err)
	}
	if err := brotliWriter.Close(); err != nil {
		t.Fatalf("Failed to close brotli writer: %v", err)
	}

	// Build realistic HTTP response
	raw := bytes.Buffer{}
	raw.WriteString("HTTP/1.1 200 OK\r\n")
	raw.WriteString("Server: nginx\r\n")
	raw.WriteString("Date: Sun, 16 Nov 2025 14:08:08 GMT\r\n")
	raw.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	raw.WriteString("Content-Encoding: br\r\n")
	raw.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(brotliBuf.Bytes())))
	raw.WriteString("\r\n")
	raw.Write(brotliBuf.Bytes())

	resp, err := response.Parse(raw.Bytes())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify response properties
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.GetServer() != "nginx" {
		t.Errorf("Expected server 'nginx', got '%s'", resp.GetServer())
	}

	// Verify decompression
	if !resp.Compressed {
		t.Error("Expected Compressed=true for brotli HTML response")
	}

	if !bytes.Equal(resp.Body, originalHTML) {
		t.Errorf("HTML not decompressed correctly.\nExpected:\n%s\n\nGot:\n%s",
			string(originalHTML), string(resp.Body))
	}
}


// ==================== RESPONSE FORMAT PRESERVATION TESTS ====================

func TestResponseParse_PreserveHeaderFormat(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type:  application/json  \r\nX-Custom:value\r\n\r\n{\"test\":true}")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Get returns original values (with whitespace preserved)
	if got := resp.Headers.Get("Content-Type"); got != "  application/json  " {
		t.Errorf("Get(\"Content-Type\") expected \"  application/json  \", got \"%s\"", got)
	}

	if got := resp.Headers.Get("X-Custom"); got != "value" {
		t.Errorf("Get(\"X-Custom\") expected \"value\", got \"%s\"", got)
	}

	// Build headers should preserve format
	headerBytes := resp.Headers.Build()
	expectedHeaders := "Content-Type:  application/json  \r\nX-Custom:value\r\n"
	if string(headerBytes) != expectedHeaders {
		t.Errorf("Header format not preserved:\nExpected: %q\nGot: %q", expectedHeaders, headerBytes)
	}
}

func TestResponseParse_PreserveLineEndings(t *testing.T) {
	// Test LF only line endings
	raw := []byte("HTTP/1.1 200 OK\nContent-Type: application/json\nServer: test\n\n{\"test\":true}")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	headerBytes := resp.Headers.Build()
	expectedHeaders := "Content-Type: application/json\nServer: test\n"
	if string(headerBytes) != expectedHeaders {
		t.Errorf("LF line endings not preserved:\nExpected: %q\nGot: %q", expectedHeaders, headerBytes)
	}
}

func TestResponseParse_MixedLineEndings(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\nX-Custom: value\r\n\r\n")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	headerBytes := resp.Headers.Build()
	expectedHeaders := "Content-Type: application/json\nX-Custom: value\r\n"
	if string(headerBytes) != expectedHeaders {
		t.Errorf("Mixed line endings not preserved:\nExpected: %q\nGot: %q", expectedHeaders, headerBytes)
	}
}

func TestResponseBuild_PreservesFormat(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\nContent-Type:  application/json  \nX-Custom:no-space\n\n{\"data\":\"test\"}")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	rebuilt := resp.Build()
	expected := "HTTP/1.1 200 OK\nContent-Type:  application/json  \nX-Custom:no-space\n\n{\"data\":\"test\"}"
	if string(rebuilt) != expected {
		t.Errorf("Response rebuild failed:\nExpected: %q\nGot: %q", expected, rebuilt)
	}
}

func TestResponseParse_ModifyHeaderClearsFormat(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type:  application/json  \r\n\r\n")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Modify the header
	resp.Headers.Set("Content-Type", "text/html")

	// Build should use standard format for modified header
	headerBytes := resp.Headers.Build()
	expected := "Content-Type: text/html\r\n"
	if string(headerBytes) != expected {
		t.Errorf("Modified header should use standard format:\nExpected: %q\nGot: %q", expected, headerBytes)
	}
}

func TestResponseParse_ComplexFormatPreservation(t *testing.T) {
	raw := []byte("HTTP/1.1 200 OK\r\n" +
		"Content-Type:application/json\r\n" +
		"Server:  nginx  \r\n" +
		"X-Tab:\tvalue\r\n" +
		"X-Empty:\r\n" +
		"\r\n" +
		"body content")

	resp, err := response.Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify parsed values (original, untrimmed)
	expectations := map[string]string{
		"Content-Type": "application/json",
		"Server":       "  nginx  ",
		"X-Tab":        "\tvalue",
		"X-Empty":      "",
	}

	for name, expected := range expectations {
		if got := resp.Headers.Get(name); got != expected {
			t.Errorf("Get(%s) expected %q, got %q", name, expected, got)
		}
	}

	// Verify original format preserved
	headerBytes := resp.Headers.Build()
	expectedHeaders := "Content-Type:application/json\r\n" +
		"Server:  nginx  \r\n" +
		"X-Tab:\tvalue\r\n" +
		"X-Empty:\r\n"

	if string(headerBytes) != expectedHeaders {
		t.Errorf("Complex format not preserved:\nExpected: %q\nGot: %q", expectedHeaders, headerBytes)
	}
}

// ============================================================================
// ParseReader Tests
// ============================================================================

func TestResponseParseReader_Basic(t *testing.T) {
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
Server: nginx

{"message":"success"}`)

	reader := bytes.NewReader(raw)
	resp, err := response.ParseReader(reader)
	if err != nil {
		t.Fatalf("ParseReader failed: %v", err)
	}

	if resp.Version != "HTTP/1.1" {
		t.Errorf("Expected version HTTP/1.1, got %s", resp.Version)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	expectedBody := `{"message":"success"}`
	if string(resp.Body) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(resp.Body))
	}
}

func TestResponseParseReader_WithOptions(t *testing.T) {
	raw := []byte(`HTTP/1.1 200 OK
Content-Type: application/json
Transfer-Encoding: chunked

5
hello
5
world
0

`)

	reader := bytes.NewReader(raw)
	opts := response.ParseOptions{
		AutoDecodeChunked: true,
	}

	resp, err := response.ParseReaderWithOptions(reader, opts)
	if err != nil {
		t.Fatalf("ParseReaderWithOptions failed: %v", err)
	}

	expectedBody := "helloworld"
	if string(resp.Body) != expectedBody {
		t.Errorf("Expected decoded body '%s', got '%s'", expectedBody, string(resp.Body))
	}

	if resp.IsBodyChunked {
		t.Error("Expected IsBodyChunked=false after auto-decode")
	}
}

func TestResponseParseReader_EmptyReader(t *testing.T) {
	reader := bytes.NewReader([]byte{})
	_, err := response.ParseReader(reader)
	if err == nil {
		t.Error("Expected error for empty reader")
	}
}

// errorReader simulates a reader that returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestResponseParseReader_ReaderError(t *testing.T) {
	reader := &errorReader{}
	_, err := response.ParseReader(reader)
	if err == nil {
		t.Error("Expected error when reader fails")
	}
}

func TestResponseParseReader_GzipCompressed(t *testing.T) {
	originalBody := []byte(`{"message":"compressed data"}`)

	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	if _, err := gzipWriter.Write(originalBody); err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	raw := bytes.Buffer{}
	raw.WriteString("HTTP/1.1 200 OK\r\n")
	raw.WriteString("Content-Encoding: gzip\r\n")
	raw.WriteString("\r\n")
	raw.Write(gzipBuf.Bytes())

	reader := bytes.NewReader(raw.Bytes())
	resp, err := response.ParseReader(reader)
	if err != nil {
		t.Fatalf("ParseReader failed: %v", err)
	}

	if !resp.Compressed {
		t.Error("Expected Compressed=true")
	}

	if !bytes.Equal(resp.Body, originalBody) {
		t.Errorf("Body not decompressed correctly")
	}
}

// ============================================================================
// Streaming Support Tests (ParseHeadersFromReader, WriteTo)
// ============================================================================

func TestResponseParseHeadersFromReader_Basic(t *testing.T) {
	raw := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 21\r\n\r\n{\"message\":\"success\"}"

	reader := strings.NewReader(raw)
	resp, bodyReader, err := response.ParseHeadersFromReader(reader)
	if err != nil {
		t.Fatalf("ParseHeadersFromReader failed: %v", err)
	}

	// Check headers are parsed
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.GetContentType() != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.GetContentType())
	}

	// Check body is NOT read yet (should be in bodyReader)
	if len(resp.Body) != 0 {
		t.Errorf("Expected empty body in response, got %d bytes", len(resp.Body))
	}

	// Read body from bodyReader
	bodyData, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	expectedBody := `{"message":"success"}`
	if string(bodyData) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(bodyData))
	}
}

func TestResponseParseHeadersFromReader_LargeBodySimulation(t *testing.T) {
	// Simulate a large body scenario
	headerPart := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 1000000\r\n\r\n"
	bodyPart := strings.Repeat("X", 1000) // Simulate part of a large body

	reader := strings.NewReader(headerPart + bodyPart)
	resp, bodyReader, err := response.ParseHeadersFromReader(reader)
	if err != nil {
		t.Fatalf("ParseHeadersFromReader failed: %v", err)
	}

	// Headers should be parsed
	if resp.GetContentLength() != 1000000 {
		t.Errorf("Expected Content-Length 1000000, got %d", resp.GetContentLength())
	}

	// Body should be streamable
	bodyData, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	if len(bodyData) != 1000 {
		t.Errorf("Expected 1000 bytes, got %d", len(bodyData))
	}
}

func TestResponseWriteTo_Basic(t *testing.T) {
	resp := response.NewResponse()
	resp.Version = "HTTP/1.1"
	resp.StatusCode = 200
	resp.StatusText = "OK"
	resp.LineSeparator = "\r\n"
	resp.Headers.Set("Content-Type", "application/json")
	resp.Body = []byte(`{"test":"data"}`)

	var buf bytes.Buffer
	n, err := resp.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if n != int64(buf.Len()) {
		t.Errorf("Bytes written mismatch: returned %d, actual %d", n, buf.Len())
	}

	// Verify the output can be parsed back
	parsed, err := response.Parse(buf.Bytes())
	if err != nil {
		t.Fatalf("Failed to parse written response: %v", err)
	}

	if parsed.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", parsed.StatusCode)
	}

	if string(parsed.Body) != `{"test":"data"}` {
		t.Errorf("Body mismatch after WriteTo")
	}
}

func TestResponseWriteHeadersTo_Basic(t *testing.T) {
	resp := response.NewResponse()
	resp.Version = "HTTP/1.1"
	resp.StatusCode = 200
	resp.StatusText = "OK"
	resp.LineSeparator = "\r\n"
	resp.Headers.Set("Content-Type", "text/plain")
	resp.Headers.Set("Content-Length", "5")

	var buf bytes.Buffer
	n, err := resp.WriteHeadersTo(&buf)
	if err != nil {
		t.Fatalf("WriteHeadersTo failed: %v", err)
	}

	if n != int64(buf.Len()) {
		t.Errorf("Bytes written mismatch")
	}

	output := buf.String()
	if !strings.HasPrefix(output, "HTTP/1.1 200 OK\r\n") {
		t.Errorf("Status line not correct: %s", output)
	}

	if !strings.Contains(output, "Content-Type: text/plain") {
		t.Errorf("Content-Type header missing")
	}
}

func TestResponseStreamingRoundTrip(t *testing.T) {
	// Create a response, write it to a buffer, then parse headers from it
	originalResp := response.NewResponse()
	originalResp.Version = "HTTP/1.1"
	originalResp.StatusCode = 200
	originalResp.StatusText = "OK"
	originalResp.LineSeparator = "\r\n"
	originalResp.Headers.Set("Content-Type", "application/octet-stream")
	originalResp.Headers.Set("Content-Length", "100")
	originalResp.Body = bytes.Repeat([]byte("A"), 100)

	// Write to buffer
	var buf bytes.Buffer
	_, err := originalResp.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	// Parse headers from buffer
	parsedResp, bodyReader, err := response.ParseHeadersFromReader(&buf)
	if err != nil {
		t.Fatalf("ParseHeadersFromReader failed: %v", err)
	}

	// Verify headers
	if parsedResp.StatusCode != originalResp.StatusCode {
		t.Errorf("Status code mismatch")
	}

	if parsedResp.GetContentType() != "application/octet-stream" {
		t.Errorf("Content-Type mismatch")
	}

	// Stream body
	bodyData, _ := io.ReadAll(bodyReader)
	if len(bodyData) != 100 {
		t.Errorf("Body length mismatch: expected 100, got %d", len(bodyData))
	}
}

// ============================================================================
// StreamingBody Tests
// ============================================================================

func TestResponseWrapBodyReader_PlainBody(t *testing.T) {
	// Create response without compression or chunking
	resp := response.NewResponse()
	resp.Headers.Set("Content-Type", "text/plain")

	bodyData := []byte("Hello, World! This is a test body.")
	bodyReader := bytes.NewReader(bodyData)

	streamBody, err := resp.WrapBodyReader(bodyReader)
	if err != nil {
		t.Fatalf("WrapBodyReader failed: %v", err)
	}
	defer streamBody.Close()

	// Read all data
	data, err := streamBody.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(data, bodyData) {
		t.Errorf("Body mismatch: expected %s, got %s", string(bodyData), string(data))
	}
}

func TestResponseWrapBodyReader_GzipDecompression(t *testing.T) {
	// Create gzip compressed body
	originalBody := []byte("This is the original body content that will be compressed with gzip.")

	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	if _, err := gzipWriter.Write(originalBody); err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}
	gzipWriter.Close()

	// Create response with gzip encoding
	resp := response.NewResponse()
	resp.Headers.Set("Content-Encoding", "gzip")

	bodyReader := bytes.NewReader(gzipBuf.Bytes())

	streamBody, err := resp.WrapBodyReader(bodyReader)
	if err != nil {
		t.Fatalf("WrapBodyReader failed: %v", err)
	}
	defer streamBody.Close()

	// Should report compressed
	if !streamBody.IsCompressed() {
		t.Error("Expected IsCompressed=true")
	}

	// Read all data - should be decompressed
	data, err := streamBody.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(data, originalBody) {
		t.Errorf("Decompression failed: expected %s, got %s", string(originalBody), string(data))
	}
}

func TestResponseWrapBodyReader_ChunkedDecoding(t *testing.T) {
	// Create chunked body
	chunkedBody := []byte("5\r\nhello\r\n6\r\n world\r\n0\r\n\r\n")

	// Create response with chunked encoding
	resp := response.NewResponse()
	resp.IsBodyChunked = true

	bodyReader := bytes.NewReader(chunkedBody)

	streamBody, err := resp.WrapBodyReader(bodyReader)
	if err != nil {
		t.Fatalf("WrapBodyReader failed: %v", err)
	}
	defer streamBody.Close()

	// Should report chunked
	if !streamBody.IsChunked() {
		t.Error("Expected IsChunked=true")
	}

	// Read all data - should be decoded
	data, err := streamBody.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	expectedBody := "hello world"
	if string(data) != expectedBody {
		t.Errorf("Chunked decoding failed: expected '%s', got '%s'", expectedBody, string(data))
	}
}

func TestResponseWrapBodyReader_ChunkedAndGzip(t *testing.T) {
	// Create gzip compressed body first
	originalBody := []byte("This body is both chunked and gzip compressed!")

	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	if _, err := gzipWriter.Write(originalBody); err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}
	gzipWriter.Close()

	// Then encode as chunked
	compressedData := gzipBuf.Bytes()
	var chunkedBuf bytes.Buffer
	chunkedBuf.WriteString(fmt.Sprintf("%x\r\n", len(compressedData)))
	chunkedBuf.Write(compressedData)
	chunkedBuf.WriteString("\r\n0\r\n\r\n")

	// Create response with both encodings
	resp := response.NewResponse()
	resp.Headers.Set("Content-Encoding", "gzip")
	resp.IsBodyChunked = true

	bodyReader := bytes.NewReader(chunkedBuf.Bytes())

	streamBody, err := resp.WrapBodyReader(bodyReader)
	if err != nil {
		t.Fatalf("WrapBodyReader failed: %v", err)
	}
	defer streamBody.Close()

	// Should report both
	if !streamBody.IsChunked() {
		t.Error("Expected IsChunked=true")
	}
	if !streamBody.IsCompressed() {
		t.Error("Expected IsCompressed=true")
	}

	// Read all data - should be decoded and decompressed
	data, err := streamBody.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(data, originalBody) {
		t.Errorf("Decoding failed: expected '%s', got '%s'", string(originalBody), string(data))
	}
}

func TestResponseStreamingBody_Search(t *testing.T) {
	// Create response with plain body
	resp := response.NewResponse()
	bodyData := []byte("The quick brown fox jumps over the lazy dog. The fox is fast.")

	bodyReader := bytes.NewReader(bodyData)
	streamBody, err := resp.WrapBodyReader(bodyReader)
	if err != nil {
		t.Fatalf("WrapBodyReader failed: %v", err)
	}
	defer streamBody.Close()

	// Search for pattern
	offset, err := streamBody.SearchString("fox")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	expectedOffset := int64(16) // Position of first "fox"
	if offset != expectedOffset {
		t.Errorf("Search offset wrong: expected %d, got %d", expectedOffset, offset)
	}
}

func TestResponseStreamingBody_SearchNotFound(t *testing.T) {
	resp := response.NewResponse()
	bodyData := []byte("Hello, World!")

	bodyReader := bytes.NewReader(bodyData)
	streamBody, err := resp.WrapBodyReader(bodyReader)
	if err != nil {
		t.Fatalf("WrapBodyReader failed: %v", err)
	}
	defer streamBody.Close()

	// Search for non-existent pattern
	offset, err := streamBody.SearchString("xyz")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if offset != -1 {
		t.Errorf("Expected -1 for not found, got %d", offset)
	}
}

func TestResponseStreamingBody_Contains(t *testing.T) {
	resp := response.NewResponse()
	bodyData := []byte("Hello, World!")

	bodyReader := bytes.NewReader(bodyData)
	streamBody, err := resp.WrapBodyReader(bodyReader)
	if err != nil {
		t.Fatalf("WrapBodyReader failed: %v", err)
	}
	defer streamBody.Close()

	found, err := streamBody.ContainsString("World")
	if err != nil {
		t.Fatalf("Contains failed: %v", err)
	}

	if !found {
		t.Error("Expected to find 'World' in body")
	}
}

func TestResponseStreamingBody_WriteTo(t *testing.T) {
	resp := response.NewResponse()
	bodyData := []byte("Stream this content to a writer.")

	bodyReader := bytes.NewReader(bodyData)
	streamBody, err := resp.WrapBodyReader(bodyReader)
	if err != nil {
		t.Fatalf("WrapBodyReader failed: %v", err)
	}
	defer streamBody.Close()

	var buf bytes.Buffer
	n, err := streamBody.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if n != int64(len(bodyData)) {
		t.Errorf("Bytes written mismatch: expected %d, got %d", len(bodyData), n)
	}

	if !bytes.Equal(buf.Bytes(), bodyData) {
		t.Errorf("Content mismatch")
	}
}

func TestResponseStreamingBody_TotalRead(t *testing.T) {
	resp := response.NewResponse()
	bodyData := []byte("Count these bytes as they are read.")

	bodyReader := bytes.NewReader(bodyData)
	streamBody, err := resp.WrapBodyReader(bodyReader)
	if err != nil {
		t.Fatalf("WrapBodyReader failed: %v", err)
	}
	defer streamBody.Close()

	// Read some data
	buf := make([]byte, 10)
	streamBody.Read(buf)

	if streamBody.TotalRead() != 10 {
		t.Errorf("TotalRead wrong: expected 10, got %d", streamBody.TotalRead())
	}

	// Read more
	streamBody.Read(buf)

	if streamBody.TotalRead() != 20 {
		t.Errorf("TotalRead wrong: expected 20, got %d", streamBody.TotalRead())
	}
}

// ============================================================================
// WriteToWithBody Tests
// ============================================================================

func TestResponseWriteToWithBody_Basic(t *testing.T) {
	resp := response.NewResponse()
	resp.Version = "HTTP/1.1"
	resp.StatusCode = 200
	resp.StatusText = "OK"
	resp.LineSeparator = "\r\n"
	resp.Headers.Set("Content-Type", "text/plain")
	resp.Headers.Set("Content-Length", "13")

	bodyData := []byte("Hello, World!")
	bodyReader := bytes.NewReader(bodyData)

	var buf bytes.Buffer
	n, err := resp.WriteToWithBody(&buf, bodyReader)
	if err != nil {
		t.Fatalf("WriteToWithBody failed: %v", err)
	}

	// Parse the result
	parsed, err := response.Parse(buf.Bytes())
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if parsed.StatusCode != 200 {
		t.Errorf("Status code wrong")
	}

	if string(parsed.Body) != "Hello, World!" {
		t.Errorf("Body wrong: got '%s'", string(parsed.Body))
	}

	if n <= int64(len(bodyData)) {
		t.Errorf("Bytes written too small: %d", n)
	}
}

func TestResponseWriteToWithBody_LargeBody(t *testing.T) {
	resp := response.NewResponse()
	resp.Version = "HTTP/1.1"
	resp.StatusCode = 200
	resp.StatusText = "OK"
	resp.LineSeparator = "\r\n"
	resp.Headers.Set("Content-Type", "application/octet-stream")

	// Simulate large body (1MB)
	bodyData := bytes.Repeat([]byte("X"), 1024*1024)
	resp.Headers.Set("Content-Length", fmt.Sprintf("%d", len(bodyData)))

	bodyReader := bytes.NewReader(bodyData)

	var buf bytes.Buffer
	_, err := resp.WriteToWithBody(&buf, bodyReader)
	if err != nil {
		t.Fatalf("WriteToWithBody failed: %v", err)
	}

	// Parse the result
	parsed, err := response.Parse(buf.Bytes())
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if len(parsed.Body) != len(bodyData) {
		t.Errorf("Body size wrong: expected %d, got %d", len(bodyData), len(parsed.Body))
	}
}

func TestResponseWriteToWithBodyChunked_Basic(t *testing.T) {
	resp := response.NewResponse()
	resp.Version = "HTTP/1.1"
	resp.StatusCode = 200
	resp.StatusText = "OK"
	resp.LineSeparator = "\r\n"
	resp.Headers.Set("Content-Type", "text/plain")

	bodyData := []byte("This is chunked encoded body content.")
	bodyReader := bytes.NewReader(bodyData)

	var buf bytes.Buffer
	_, err := resp.WriteToWithBodyChunked(&buf, bodyReader, 10) // Small chunk size for testing
	if err != nil {
		t.Fatalf("WriteToWithBodyChunked failed: %v", err)
	}

	// Parse the result with auto-decode
	opts := response.ParseOptions{AutoDecodeChunked: true}
	parsed, err := response.ParseWithOptions(buf.Bytes(), opts)
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if string(parsed.Body) != string(bodyData) {
		t.Errorf("Body wrong: expected '%s', got '%s'", string(bodyData), string(parsed.Body))
	}
}

func TestResponseWriteToWithBodyChunked_LargeBody(t *testing.T) {
	resp := response.NewResponse()
	resp.Version = "HTTP/1.1"
	resp.StatusCode = 200
	resp.StatusText = "OK"
	resp.LineSeparator = "\r\n"
	resp.Headers.Set("Content-Type", "application/octet-stream")

	// Large body (100KB)
	bodyData := bytes.Repeat([]byte("Y"), 100*1024)
	bodyReader := bytes.NewReader(bodyData)

	var buf bytes.Buffer
	_, err := resp.WriteToWithBodyChunked(&buf, bodyReader, 8192) // Default chunk size
	if err != nil {
		t.Fatalf("WriteToWithBodyChunked failed: %v", err)
	}

	// Parse the result with auto-decode
	opts := response.ParseOptions{AutoDecodeChunked: true}
	parsed, err := response.ParseWithOptions(buf.Bytes(), opts)
	if err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if len(parsed.Body) != len(bodyData) {
		t.Errorf("Body size wrong: expected %d, got %d", len(bodyData), len(parsed.Body))
	}

	if !bytes.Equal(parsed.Body, bodyData) {
		t.Errorf("Body content mismatch")
	}
}

// ============================================================================
// Full Streaming Pipeline Test
// ============================================================================

func TestResponseFullStreamingPipeline(t *testing.T) {
	// Test the full pipeline: parse headers -> wrap body -> search -> stream out

	// Create a gzip compressed response
	originalBody := []byte("This is a test body with searchable content: MARKER_123 found here!")

	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	gzipWriter.Write(originalBody)
	gzipWriter.Close()

	// Build full HTTP response
	var rawResponse bytes.Buffer
	rawResponse.WriteString("HTTP/1.1 200 OK\r\n")
	rawResponse.WriteString("Content-Type: text/plain\r\n")
	rawResponse.WriteString("Content-Encoding: gzip\r\n")
	rawResponse.WriteString(fmt.Sprintf("Content-Length: %d\r\n", gzipBuf.Len()))
	rawResponse.WriteString("\r\n")
	rawResponse.Write(gzipBuf.Bytes())

	// Step 1: Parse headers from reader
	resp, bodyReader, err := response.ParseHeadersFromReader(bytes.NewReader(rawResponse.Bytes()))
	if err != nil {
		t.Fatalf("ParseHeadersFromReader failed: %v", err)
	}

	// Verify headers parsed correctly
	if resp.StatusCode != 200 {
		t.Errorf("Status code wrong: %d", resp.StatusCode)
	}

	if resp.GetContentEncoding() != "gzip" {
		t.Errorf("Content-Encoding wrong: %s", resp.GetContentEncoding())
	}

	// Step 2: Wrap body reader for decompression
	streamBody, err := resp.WrapBodyReader(bodyReader)
	if err != nil {
		t.Fatalf("WrapBodyReader failed: %v", err)
	}
	defer streamBody.Close()

	// Step 3: Search for marker in decompressed stream
	offset, err := streamBody.SearchString("MARKER_123")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if offset < 0 {
		t.Error("Marker not found in decompressed stream")
	}
}

func TestResponseStreamingBody_BrotliDecompression(t *testing.T) {
	// Create brotli compressed body
	originalBody := []byte("This is the original body content that will be compressed with brotli.")

	var brotliBuf bytes.Buffer
	brotliWriter := brotli.NewWriter(&brotliBuf)
	if _, err := brotliWriter.Write(originalBody); err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}
	brotliWriter.Close()

	// Create response with brotli encoding
	resp := response.NewResponse()
	resp.Headers.Set("Content-Encoding", "br")

	bodyReader := bytes.NewReader(brotliBuf.Bytes())

	streamBody, err := resp.WrapBodyReader(bodyReader)
	if err != nil {
		t.Fatalf("WrapBodyReader failed: %v", err)
	}
	defer streamBody.Close()

	// Should report compressed
	if !streamBody.IsCompressed() {
		t.Error("Expected IsCompressed=true")
	}

	// Read all data - should be decompressed
	data, err := streamBody.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(data, originalBody) {
		t.Errorf("Brotli decompression failed: expected %s, got %s", string(originalBody), string(data))
	}
}

